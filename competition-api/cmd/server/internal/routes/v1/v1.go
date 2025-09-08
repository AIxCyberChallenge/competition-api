package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges"
	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/github"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs"
	servermiddleware "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/ratelimit"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/taskrunner"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/server/routes/v1"

var tracer = otel.Tracer(name)

type Handler struct {
	DB *gorm.DB
	// If not nil jobs will be queued.
	// TODO: figure out kind in tests or something such that tests can queue jobs
	// OR: Mock the client and remove the null checks
	JobClient          *jobs.KubernetesClient
	githubClient       *github.Client
	taskrunnerClient   *taskrunner.Client
	challengesClient   *challenges.Client
	config             *config.Config
	archiver           upload.Uploader
	submissionUploader upload.Uploader
	sourcesUploader    upload.Uploader
}

func NewRedisLimiter(
	redisHost string,
	limiterKey string,
	perMinute int64,
	failOpen bool,
	onlyMethod *string,
) middleware.RateLimiterConfig {
	l := logger.Logger
	var store middleware.RateLimiterStore

	redisAddr := redisHost + ":6379"
	l.Debug("Setting up rate limiter with Redis", "redis", redisAddr)
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	rdConf := &ratelimit.RedisLimiterConfig{
		PerMinute:   perMinute,
		RedisClient: rdb,
		LimiterKey:  limiterKey,
		FailOpen:    failOpen,
	}
	store = ratelimit.NewRedisLimitStore(*rdConf)

	skipper := middleware.DefaultSkipper
	if onlyMethod != nil {
		skipper = func(c echo.Context) bool {
			return c.Request().Method != *onlyMethod
		}
	}

	return middleware.RateLimiterConfig{
		Skipper: skipper,
		Store:   store,
		IdentifierExtractor: func(c echo.Context) (string, error) {
			auth, ok := c.Get("auth").(*models.Auth)
			if !ok {
				return "", srverr.ErrTypeAssertMismatch
			}
			return auth.ID.String(), nil
		},
		ErrorHandler: func(context echo.Context, _ error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, _ string, _ error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}
}

func NewHandler(
	db *gorm.DB,
	jobClient *jobs.KubernetesClient,
	githubClient *github.Client,
	taskrunnerClient *taskrunner.Client,
	challengesClient *challenges.Client,
	cfg *config.Config,
	archiver upload.Uploader,
	submissionUploader upload.Uploader,
	sourcesUploader upload.Uploader,
) Handler {
	return Handler{
		DB:                 db,
		JobClient:          jobClient,
		taskrunnerClient:   taskrunnerClient,
		githubClient:       githubClient,
		challengesClient:   challengesClient,
		config:             cfg,
		archiver:           archiver,
		submissionUploader: submissionUploader,
		sourcesUploader:    sourcesUploader,
	}
}

func (h *Handler) AddRoutes(e *echo.Echo, middlewareHandler *servermiddleware.Handler) {
	l := logger.Logger

	v1Group := e.Group("/v1", middleware.BasicAuth(middlewareHandler.BasicAuthValidator))

	if h.config.RateLimit != nil && h.config.RateLimit.GlobalPerMinute > 0 {
		v1Group.Use(
			middleware.RateLimiterWithConfig(
				NewRedisLimiter(
					h.config.RateLimit.RedisHost,
					"global",
					h.config.RateLimit.GlobalPerMinute,
					h.config.RateLimit.FailOpen,
					nil,
				),
			),
		)
	} else {
		l.Warn("not configured to have a global rate limit")
	}

	v1Group.GET("/ping/", h.Ping)

	taskGroup := v1Group.Group(
		"/task/:task_id",
		servermiddleware.HasPermissions("auth", &models.Permissions{CRS: true}),
		servermiddleware.PopulateFromIDParam[models.Task](middlewareHandler, "task_id", "task"),
		servermiddleware.RoundID(
			map[string]bool{*h.config.RoundID: true, *h.config.Generate.RoundID: true},
			"task",
		),
	)
	requestGroup := v1Group.Group(
		"/request",
		servermiddleware.HasPermissions("auth", &models.Permissions{CRS: true}),
	)

	submittedSARIFGroup := taskGroup.Group("/submitted-sarif")
	povGroup := taskGroup.Group("/pov")
	patchGroup := taskGroup.Group("/patch")

	if h.config.RateLimit != nil && h.config.RateLimit.SubmitPerMinute > 0 {
		post := http.MethodPost

		submittedSARIFGroup.Use(
			middleware.RateLimiterWithConfig(
				NewRedisLimiter(
					h.config.RateLimit.RedisHost,
					"global",
					h.config.RateLimit.SubmitPerMinute,
					h.config.RateLimit.FailOpen,
					&post,
				),
			),
		)
		povGroup.Use(
			middleware.RateLimiterWithConfig(
				NewRedisLimiter(
					h.config.RateLimit.RedisHost,
					"global",
					h.config.RateLimit.SubmitPerMinute,
					h.config.RateLimit.FailOpen,
					&post,
				),
			),
		)
		patchGroup.Use(
			middleware.RateLimiterWithConfig(
				NewRedisLimiter(
					h.config.RateLimit.RedisHost,
					"global",
					h.config.RateLimit.SubmitPerMinute,
					h.config.RateLimit.FailOpen,
					&post,
				),
			),
		)
	} else {
		l.Warn("not configured to have a submit rate limit")
	}

	broadcastSARIFGroup := taskGroup.Group("/broadcast-sarif-assessment")
	bundleGroup := taskGroup.Group("/bundle")
	freeformGroup := taskGroup.Group("/freeform")

	broadcastSARIFGroup.POST(
		"/:sarif_id/",
		h.SubmitSarifAssessment,
		servermiddleware.PopulateFromIDParam[models.SARIFBroadcast](
			middlewareHandler,
			"sarif_id",
			"sarif",
		),
	)

	submittedSARIFGroup.POST("/", h.SubmitSarif)

	povGroup.POST("/", h.SubmitPOV)
	povGroup.GET(
		"/:pov_id/",
		h.POVStatus,
		servermiddleware.PopulateFromIDParam[models.POVSubmission](
			middlewareHandler,
			"pov_id",
			"pov",
		),
	)

	patchGroup.POST("/", h.SubmitPatch)
	patchGroup.GET(
		"/:patch_id/",
		h.PatchStatus,
		servermiddleware.PopulateFromIDParam[models.PatchSubmission](
			middlewareHandler,
			"patch_id",
			"patch",
		),
	)

	bundleGroup.POST("/", h.SubmitBundle)
	bundleGroup.GET(
		"/:bundle_id/",
		h.GetBundle,
		servermiddleware.PopulateFromIDParam[models.Bundle](
			middlewareHandler,
			"bundle_id",
			"bundle",
		),
	)
	bundleGroup.PATCH(
		"/:bundle_id/",
		h.PatchBundle,
		servermiddleware.PopulateFromIDParam[models.Bundle](
			middlewareHandler,
			"bundle_id",
			"bundle",
		),
	)
	bundleGroup.DELETE(
		"/:bundle_id/",
		h.DeleteBundle,
		servermiddleware.PopulateFromIDParam[models.Bundle](
			middlewareHandler,
			"bundle_id",
			"bundle",
		),
	)

	freeformGroup.POST("/", h.SubmitFreeform)

	if *h.config.Generate.Enabled {
		requestGroup.GET("/list/", h.RequestList)
		requestGroup.POST("/:challenge_name/", h.RequestChallenge)
	}
}
