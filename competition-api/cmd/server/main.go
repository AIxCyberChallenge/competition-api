package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	sloggorm "github.com/orandin/slog-gorm"
	"github.com/sethvargo/go-retry"
	otellib "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/github"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs"
	servermiddleware "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/migrations"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/competition"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/jobrunner"
	routesv1 "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/v1"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/webhooks"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/taskrunner"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/otel"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

const name string = "github.com/aixcyberchallenge/competition-api/competition-api/server"

var tracer = otellib.Tracer(name)

type server struct {
	archiver                       upload.Uploader
	router                         *echo.Echo
	config                         *config.Config
	db                             *gorm.DB
	taskRunner                     *taskrunner.Client
	otelShutdown                   func(context.Context) error
	competitionAPIController       *jobs.CompetitionAPIController
	competitionAPIControllerCancel func()
	jobClient                      jobs.KubernetesClient
}

func initServer(ctx context.Context) (*server, error) {
	server := new(server)

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server config: %w", err)
	}
	server.config = cfg

	shutdownOTel, err := otel.SetupOTelSDK(ctx, cfg.Logging.UseOTLP)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OTEL SDK: %w", err)
	}
	defer func() {
		// Something failed to initialize, make sure everything gets flushed to the server
		if server.otelShutdown == nil {
			otelShutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*time.Duration(cfg.GracefulShutdownSecs),
			)
			defer cancel()

			if err = shutdownOTel(otelShutdownCtx); err != nil {
				logger.Logger.Error("failed to flush otel data", "error", err)
			}
		}
	}()

	ctx, span := tracer.Start(ctx, "initServer")
	defer span.End()

	logger.LogLevel.Set(slog.Level(cfg.Logging.App.Level))
	gormLogger := slog.New(logger.Handler)

	sg := sloggorm.New(
		sloggorm.WithHandler(gormLogger.Handler()),
		sloggorm.SetLogLevel(sloggorm.DefaultLogType, slog.Level(cfg.Logging.Gorm.Level)),
	)
	if cfg.Logging.Gorm.TraceQueries {
		sg = sloggorm.New(
			sloggorm.WithHandler(gormLogger.Handler()),
			sloggorm.WithTraceAll(),
			sloggorm.SetLogLevel(sloggorm.DefaultLogType, slog.Level(cfg.Logging.Gorm.Level)),
		)
	}

	span.AddEvent("initialized gorm logging")

	db, err := gorm.Open(
		postgres.Open(cfg.PostgresDSN()),
		&gorm.Config{Logger: sg, TranslateError: true},
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to initialize database")
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to acquire underlying database connection")
		return nil, fmt.Errorf("failed to acquire underlying database connection: %w", err)
	}

	// Configure db connection pool
	sqlDB.SetMaxIdleConns(cfg.Postgres.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(cfg.Postgres.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(cfg.Postgres.ConnectionTTL)

	span.AddEvent("initialized database connection")

	err = db.Use(gormtracing.NewPlugin())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to add otel plugin to gorm")
		return nil, fmt.Errorf("failed to add otel plugin to gorm: %w", err)
	}

	span.AddEvent("added the otel plugin to gorm")

	err = migrations.Up(ctx, db)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to preform database migrations")
		return nil, fmt.Errorf("failed to perform database migrations: %w", err)
	}

	span.AddEvent("migrated database to latest version")

	azureCred, err := azblob.NewSharedKeyCredential(
		cfg.Azure.StorageAccount.Name,
		cfg.Azure.StorageAccount.Key,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to initialize azure credentials")
		return nil, fmt.Errorf("failed to initialize azure credentials: %w", err)
	}

	span.AddEvent("initialized azure storage account credentials")

	azureClient, err := azblob.NewClientWithSharedKeyCredential(
		cfg.Azure.StorageAccount.Containers.URL,
		azureCred,
		nil,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to initialize azure client")
		return nil, fmt.Errorf("failed to initialize azure client: %w", err)
	}

	span.AddEvent("initialized azure storage account")

	if err = models.LoadAPIKeysFromConfig(ctx, db, cfg.Teams); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load API keys from config")
		return nil, fmt.Errorf("failed to load API keys from config: %w", err)
	}

	span.AddEvent("loaded api keys from config")

	if cfg.Azure.Dev {
		if err = setupContainers(ctx, azureClient, cfg.Azure.StorageAccount.Containers); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error setting up containers for dev environment")
			return nil, fmt.Errorf("error setting up containers for dev environment: %w", err)
		}
	}

	var clusterConfig *rest.Config
	if cfg.K8s.InCluster {
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error fetching in cluster config")
			return nil, fmt.Errorf("error fetching in cluster config: %w", err)
		}
	} else {
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", homedir.HomeDir()+"/.kube/config")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error fetching in home dir cluster config")
			return nil, fmt.Errorf("error fetching in home dir cluster config: %w", err)
		}
	}

	span.AddEvent("got k8s cluster config")

	clusterConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 3
		retryClient.RetryWaitMin = 100 * time.Millisecond
		retryClient.RetryWaitMax = 5 * time.Second
		retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
			if k8serrs.IsNotFound(err) {
				// don't retry on not found
				return false, nil
			}

			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}
		// Use transport from standard client since retry logic is wrapped into it
		retryClient.HTTPClient.Transport = rt
		return retryClient.StandardClient().Transport
	})

	k8sClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error creating k8s client from cluster config")
		return nil, fmt.Errorf("error creating k8s client from cluster config: %w", err)
	}

	server.competitionAPIController, err = jobs.NewCompetitionAPIController(
		k8sClient,
		cfg.K8s.Namespace,
		uuid.New().String(),
		db,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create competitionapi controller")
		return nil, err
	}

	span.AddEvent("initialized k8s client")

	taskRunnerClient := taskrunner.Create()
	jobClient := jobs.CreateJobClient(
		cfg.K8s.Namespace,
		k8sClient,
		cfg,
	)

	archiver, err := upload.NewMinioUploader(
		cfg.S3Archive.Endpoint,
		cfg.S3Archive.AccessKeyID,
		cfg.S3Archive.SecretAccessKey,
		cfg.S3Archive.SSLEnabled,
		cfg.S3Archive.BucketName,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to construct archiver")
		return nil, err
	}

	sourcesUploader := upload.NewAzureUploaderFromClient(
		azureClient,
		cfg.Azure.StorageAccount.Containers.Sources,
	)
	submissionUploader := upload.NewAzureUploaderFromClient(
		azureClient,
		cfg.Azure.StorageAccount.Containers.Submissions,
	)
	artifactsUploader := upload.NewAzureUploaderFromClient(
		azureClient,
		cfg.Azure.StorageAccount.Containers.Artifacts,
	)
	challengesClient := challenges.Create(
		db,
		*cfg.TempDir,
		&jobClient,
		upload.NewRetryUploader(archiver),
		upload.NewRetryUploader(sourcesUploader),
	)
	githubClient, err := github.Create(cfg.Github)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to construct github client")
		return nil, fmt.Errorf("failed to construct github client: %w", err)
	}

	span.AddEvent("initialized github client")

	webhookHandler := webhooks.CreateHandler(
		db,
		taskRunnerClient,
		challengesClient,
		*cfg.RoundID,
		githubClient,
		cfg.Teams,
		cfg.IgnoredRepos,
		upload.NewRetryUploader(archiver),
	)

	backoff := func() retry.Backoff {
		b := retry.NewFibonacci(time.Millisecond * 25)
		b = retry.WithMaxRetries(3, b)
		return b
	}
	v1Handler := routesv1.NewHandler(
		db,
		&jobClient,
		githubClient,
		taskRunnerClient,
		challengesClient,
		cfg,
		upload.NewRetryUploaderBackoff(archiver, backoff),
		upload.NewRetryUploaderBackoff(submissionUploader, backoff),
		upload.NewRetryUploaderBackoff(sourcesUploader, backoff),
	)
	competitionHandler := competition.Create(cfg, &jobClient, db)
	middlewareHandler := servermiddleware.Handler{DB: db}
	jobrunnerHandler := jobrunner.NewHandler(
		db,
		&jobClient,
		taskRunnerClient,
		cfg,
		upload.NewRetryUploaderBackoff(submissionUploader, backoff),
		upload.NewRetryUploaderBackoff(artifactsUploader, backoff),
		upload.NewRetryUploaderBackoff(sourcesUploader, backoff),
	)

	e, err := routes.BuildEcho(logger.Logger)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error building router")
		return nil, fmt.Errorf("error building router: %w", err)
	}

	span.AddEvent("created echo router")

	v1Handler.AddRoutes(e, &middlewareHandler)
	webhookHandler.AddRoutes(e)
	competitionHandler.AddRoutes(e, &middlewareHandler)
	jobrunnerHandler.AddRoutes(e, &middlewareHandler)

	server.otelShutdown = shutdownOTel
	server.router = e
	server.jobClient = jobClient
	server.db = db
	server.taskRunner = taskRunnerClient
	server.archiver = archiver

	return server, nil
}

func (s *server) Start(ctx context.Context) error {
	qr, err := queue.NewAzureQueuer(
		s.config.Azure.StorageAccount.Name,
		s.config.Azure.StorageAccount.Key,
		s.config.Azure.StorageAccount.Queues.URL,
		s.config.Azure.StorageAccount.Queues.Results,
	)
	if err != nil {
		return err
	}

	fetcher, err := fetch.NewAzureFetcher(
		s.config.Azure.StorageAccount.Name,
		s.config.Azure.StorageAccount.Key,
		s.config.Azure.StorageAccount.Containers.URL,
		s.config.Azure.StorageAccount.Containers.Artifacts,
	)
	if err != nil {
		return err
	}

	// TODO: make this shutdown gracefully
	go jobs.MonitorResultsQueue(
		ctx,
		s.db,
		qr,
		upload.NewRetryUploader(s.archiver),
		fetcher,
	)

	competitionAPIControllerCtx, competitionAPIControllerCancel := context.WithCancel(ctx)
	go s.competitionAPIController.Run(competitionAPIControllerCtx)
	s.competitionAPIControllerCancel = competitionAPIControllerCancel

	logger.Logger.Info("Starting services...")

	err = s.router.Start(s.config.ListenAddress)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *server) Shutdown() error {
	var errs error

	ctx, cancelTimeout := context.WithTimeout(
		context.Background(),
		time.Second*time.Duration(s.config.GracefulShutdownSecs),
	)
	defer cancelTimeout()

	s.competitionAPIControllerCancel()

	// TODO: do we want these serialized shutdowns?
	if err := s.router.Shutdown(ctx); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := s.taskRunner.Shutdown(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to shutdown taskRunner gracefully: %w", err))
	}

	if s.otelShutdown != nil {
		errs = errors.Join(errs, s.otelShutdown(ctx))
	}

	return errs
}

// @title						Competition API
// @version					0.2
// @securityDefinitions.basic	BasicAuth
func main() {
	ctx, cancelSignal := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	logger.InitSlog()

	server, err := initServer(ctx)
	if err != nil {
		logger.Logger.Error(err.Error())
		cancelSignal()
		os.Exit(1)
	}

	errch := make(chan error, 1)
	go func() {
		<-ctx.Done()
		logger.Logger.Info("Got shutdown signal!")
		errch <- server.Shutdown()
		close(errch)
	}()

	if err := server.Start(ctx); err != nil {
		logger.Logger.Error(err.Error())
		cancelSignal()
		os.Exit(1)
	}

	if err := <-errch; err != nil {
		logger.Logger.Error("Error shutting down server", "error", err)
	}

	cancelSignal()
}

func setupContainers(
	ctx context.Context,
	azureClient *azblob.Client,
	containers *config.AzureStorageAccountContainerConfig,
) error {
	pager := azureClient.NewListContainersPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, c := range page.ContainerItems {
			_, err = azureClient.DeleteContainer(ctx, *c.Name, nil)
			if err != nil {
				return err
			}
		}
	}

	_, err := azureClient.CreateContainer(ctx, containers.Submissions, nil)
	if err != nil {
		return err
	}
	_, err = azureClient.CreateContainer(ctx, containers.Sources, nil)
	if err != nil {
		return err
	}

	return nil
}
