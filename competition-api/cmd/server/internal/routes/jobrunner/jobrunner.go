package jobrunner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs"
	servermiddleware "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/taskrunner"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/server/routes/jobs"

var tracer = otel.Tracer(name)

type Handler struct {
	DB *gorm.DB
	// If not nil jobs will be queued.
	// TODO: figure out kind in tests or something such that tests can queue jobs
	// OR: Mock the client and remove the null checks
	JobClient          *jobs.KubernetesClient
	taskrunnerClient   *taskrunner.Client
	config             *config.Config
	submissionUploader upload.Uploader
	artifactsUploader  upload.Uploader
	sourcesUploader    upload.Uploader
}

func NewHandler(
	db *gorm.DB,
	jobClient *jobs.KubernetesClient,
	taskrunnerClient *taskrunner.Client,
	cfg *config.Config,
	submissionUploader upload.Uploader,
	artifactsUploader upload.Uploader,
	sourcesUploader upload.Uploader,
) Handler {
	return Handler{
		DB:                 db,
		JobClient:          jobClient,
		taskrunnerClient:   taskrunnerClient,
		config:             cfg,
		submissionUploader: submissionUploader,
		artifactsUploader:  artifactsUploader,
		sourcesUploader:    sourcesUploader,
	}
}

func (h *Handler) AddRoutes(e *echo.Echo, middlewareHandler *servermiddleware.Handler) {
	jobsGroup := e.Group("/jobrunner",
		middleware.BasicAuth(middlewareHandler.BasicAuthValidator),
		servermiddleware.HasPermissions("auth", &models.Permissions{JobRunner: true}),
	)

	jobsGroup.GET(
		"/job/:job_id/",
		h.GetJobResults,
		servermiddleware.PopulateFromIDParam[models.Job](middlewareHandler, "job_id", "job"),
	)

	jobsGroup.POST("/job/", h.RunTest)
	jobsGroup.POST("/job/bulk/", h.RunBulkTests)
	jobsGroup.POST("/job/bulk/results/", h.PostJobResultsBulk)
}

type presignedJobArtifacts struct {
	Artifacts []types.JobArtifact
	Results   []types.JobResult
}

func (h *Handler) presignJob(ctx context.Context, job *models.Job) (*presignedJobArtifacts, error) {
	ctx, span := tracer.Start(ctx, "presignJob", trace.WithAttributes(
		attribute.String("job.id", job.ID.String()),
	))
	defer span.End()

	duration := time.Hour * 100
	presigned := &presignedJobArtifacts{
		Artifacts: make([]types.JobArtifact, 0, len(job.Artifacts)),
		Results:   make([]types.JobResult, 0, len(job.Results)),
	}

	span.AddEvent("getting presigned URLs for artifacts")
	for _, artifact := range job.Artifacts {
		url, err := h.artifactsUploader.PresignedReadURL(
			ctx,
			artifact.Blob.ObjectName,
			duration,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned URL for artifact")
			span.RecordError(err)
			return nil, err
		}

		artifact.Blob.PresignedURL = &url
		presigned.Artifacts = append(presigned.Artifacts, artifact)
	}

	span.AddEvent("getting presigned URLs for results")
	for _, result := range job.Results {
		stdoutURL, err := h.artifactsUploader.PresignedReadURL(
			ctx,
			result.StdoutBlob.ObjectName,
			duration,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned URL for stdout result")
			span.RecordError(err)
			return nil, err
		}

		stderrURL, err := h.artifactsUploader.PresignedReadURL(
			ctx,
			result.StderrBlob.ObjectName,
			duration,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned URL for stderr result")
			span.RecordError(err)
			return nil, err
		}

		result.StdoutBlob.PresignedURL = &stdoutURL
		result.StderrBlob.PresignedURL = &stderrURL
		presigned.Results = append(presigned.Results, result)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "generated presigned")
	return presigned, nil
}

//gocyclo:ignore
func (h *Handler) doJobWork(
	ctx context.Context,
	jobRequest *types.JobArgs,
) (*types.JobResponse, error) {
	ctx, span := tracer.Start(ctx, "doJobWork")
	defer span.End()

	db := h.DB.WithContext(ctx)

	task := &models.Task{}
	var taskID uuid.UUID
	sources := &models.PresignedSourceURLs{}

	cacheToHash := []string{*h.config.CacheKey, *jobRequest.CacheKey}

	if jobRequest.TaskID != nil {
		var err error
		taskID, err = uuid.Parse(*jobRequest.TaskID)
		if err != nil {
			span.SetStatus(codes.Error, "failed to parse task ID as a UUID")
			span.RecordError(err)
			return nil, echo.NewHTTPError(
				http.StatusBadRequest,
				types.StringError("failed to parse task ID as a UUID"),
			)
		}

		span.AddEvent("fetching task from db")
		task, err = models.ByID[models.Task](ctx, db, taskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				span.SetStatus(codes.Error, "TaskID not found")
				span.RecordError(err)
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					types.Error{Message: "task_id not found", Fields: &map[string]string{
						"task_id": "not found",
					}},
				)
			}
			span.SetStatus(codes.Error, "error fetching task from db")
			span.RecordError(err)
			return nil, response.InternalServerError
		}
		span.AddEvent("getting source presigned URLs")
		expiration := time.Hour * 100
		sources, err = task.GetSourceURLs(ctx, h.sourcesUploader, expiration)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned urls")
			span.RecordError(err)
			return nil, response.InternalServerError
		}
	}

	if jobRequest.ProjectName != nil {
		task.ProjectName = *jobRequest.ProjectName
		span.AddEvent("Found provided job argument: Project Name")
	}
	if jobRequest.Focus != nil {
		task.Focus = *jobRequest.Focus
		span.AddEvent("Found provided job argument: Focus")
	}
	if jobRequest.MemoryGB != nil {
		span.AddEvent("memory_gb is set, overriding memory gb with the value")
		task.MemoryGB = *jobRequest.MemoryGB
	}
	if jobRequest.CPUs != nil {
		span.AddEvent("cpus is set, overriding cpus with the value")
		task.CPUs = *jobRequest.CPUs
	}
	if jobRequest.HeadRepoTarballURL != nil {
		span.AddEvent(
			"repo_tarball_url is set, overriding Repo Tarball URL with provided URL",
		)
		sources.HeadRepo = *jobRequest.HeadRepoTarballURL
		cacheToHash = append(cacheToHash, sources.HeadRepo)
	} else {
		cacheToHash = append(cacheToHash, task.UnstrippedSource.HeadRepo.SHA256)
	}
	if jobRequest.OssFuzzTarballURL != nil {
		span.AddEvent(
			"oss_fuzz_tarball is set, overriding OSS Fuzz Tarball URL with provided URL",
		)
		sources.FuzzTooling = *jobRequest.OssFuzzTarballURL
		cacheToHash = append(cacheToHash, sources.FuzzTooling)
	} else {
		cacheToHash = append(cacheToHash, task.UnstrippedSource.FuzzTooling.SHA256)
	}
	if jobRequest.BaseTarballURL != nil {
		span.AddEvent(
			"diff_tarball is set, overriding Diff Tarball URL with provided URL",
		)
		sources.BaseRepo = *jobRequest.BaseTarballURL
		cacheToHash = append(cacheToHash, sources.BaseRepo)
	} else if task.UnstrippedSource.BaseRepo != nil {
		cacheToHash = append(cacheToHash, task.UnstrippedSource.BaseRepo.SHA256)
	}

	skipPatchTests := false
	if jobRequest.SkipPatchTests != nil {
		skipPatchTests = *jobRequest.SkipPatchTests
	}

	args := []string{
		"--head-repo-url", sources.HeadRepo,
		"--oss-fuzz-url", sources.FuzzTooling,
		"--focus", task.Focus,
		"--project-name", task.ProjectName,
		"--architecture", *jobRequest.Architecture,
		"--export-results",
	}
	cacheToHash = append(cacheToHash, task.Focus, task.ProjectName)

	if skipPatchTests {
		args = append(args, "--skip-patch-tests")
	}
	cacheToHash = append(cacheToHash, strconv.FormatBool(skipPatchTests))

	if sources.BaseRepo != "" {
		args = append(args, "--base-repo-url", sources.BaseRepo)
	}

	testcaseBlob := jobRequest.TestcaseHash
	if testcaseBlob != nil {
		exists, err := h.submissionUploader.Exists(ctx, *testcaseBlob)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to check if pov blob exists")
			return nil, response.InternalServerError
		}

		if !exists {
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "missing pov blob")
			return nil, echo.NewHTTPError(
				http.StatusUnprocessableEntity,
				types.StringError("missing pov blob"),
			)
		}
	}
	if jobRequest.TestcaseB64 != nil {
		span.AddEvent("setting up objects and args for PoV run")

		span.AddEvent("validating trigger is within size limit")
		if !validator.ValidateTriggerSize(len(*jobRequest.TestcaseB64)) {
			span.RecordError(errors.New("trigger was too large"))
			span.SetStatus(codes.Error, "trigger was too large")
			span.RecordError(nil)
			return nil, echo.NewHTTPError(
				http.StatusBadRequest,
				types.Error{Message: "validation error", Fields: &map[string]string{
					"testcase": "must be <= 2mb",
				}},
			)
		}

		span.AddEvent("decoding trigger base64")
		vulnData, err := base64.StdEncoding.DecodeString(*jobRequest.TestcaseB64)
		if err != nil {
			span.SetStatus(codes.Error, "failed to decode trigger")
			span.RecordError(err)
			return nil, echo.NewHTTPError(
				http.StatusBadRequest,
				types.Error{Message: "failed to decode base64", Fields: &map[string]string{
					"testcase": "must be valid base64",
				}},
			)
		}

		span.AddEvent("uploading trigger")
		blobName, err := upload.Hashed(
			ctx,
			h.submissionUploader,
			bytes.NewReader(vulnData),
			int64(len(vulnData)),
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to upload trigger")
			span.RecordError(err)
			return nil, response.InternalServerError
		}

		span.SetAttributes(
			attribute.String("testcase.sum", blobName),
			attribute.String("blob.name", blobName),
		)

		testcaseBlob = &blobName
	}

	if testcaseBlob != nil {
		expiration := time.Hour * 100
		span.AddEvent("getting presigned pov url", trace.WithAttributes(
			attribute.String("expiration", expiration.String()),
		))
		triggerURL, err := h.submissionUploader.PresignedReadURL(
			ctx,
			*testcaseBlob,
			expiration,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned url")
			span.RecordError(err)
			return nil, response.InternalServerError
		}
		args = append(args,
			"--trigger-url", triggerURL,
			"--sanitizer", *jobRequest.Sanitizer,
			"--harness-name", *jobRequest.FuzzerName,
			"--engine", *jobRequest.Engine,
		)
		cacheToHash = append(
			cacheToHash,
			"testcase",
			*testcaseBlob,
			*jobRequest.Architecture,
			*jobRequest.Sanitizer,
			*jobRequest.FuzzerName,
			*jobRequest.Engine,
		)
	}

	patchBlob := jobRequest.PatchHash
	if patchBlob != nil {
		exists, err := h.submissionUploader.Exists(ctx, *patchBlob)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to check if patch blob exists")
			return nil, response.InternalServerError
		}

		if !exists {
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "missing patch blob")
			return nil, echo.NewHTTPError(
				http.StatusUnprocessableEntity,
				types.StringError("missing patch blob"),
			)
		}
	}
	if jobRequest.PatchB64 != nil {
		span.AddEvent("setting up objects and args for patch run")

		span.AddEvent("validating patch is within size limit")
		if !validator.ValidatePatchSize(len(*jobRequest.PatchB64)) {
			span.SetStatus(codes.Error, "patch was too large")
			span.RecordError(errors.New("patch was too large"))
			return nil, echo.NewHTTPError(
				http.StatusBadRequest,
				types.Error{Message: "validation error", Fields: &map[string]string{
					"testcase": "must be <= 100kb",
				}},
			)
		}

		span.AddEvent("decoding patch base64")
		patchData, err := base64.StdEncoding.DecodeString(*jobRequest.PatchB64)
		if err != nil {
			span.SetStatus(codes.Error, "failed to decode patch")
			span.RecordError(err)
			return nil, echo.NewHTTPError(
				http.StatusBadRequest,
				types.Error{Message: "failed to decode base64", Fields: &map[string]string{
					"patch": "must be valid base64",
				}},
			)
		}

		span.AddEvent("uploading patch")
		blobName, err := upload.Hashed(
			ctx,
			h.submissionUploader,
			bytes.NewReader(patchData),
			int64(len(patchData)),
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to upload patch")
			span.RecordError(err)
			return nil, response.InternalServerError
		}

		patchBlob = &blobName
	}

	if patchBlob != nil {
		span.SetAttributes(
			attribute.String("patch.hash", *patchBlob),
			attribute.String("blob.name", *patchBlob),
		)

		expiration := time.Hour * 100
		span.AddEvent("getting presigned patch url", trace.WithAttributes(
			attribute.String("expiration", expiration.String()),
		))
		patchURL, err := h.submissionUploader.PresignedReadURL(
			ctx,
			*patchBlob,
			expiration,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned url")
			span.RecordError(err)
			return nil, response.InternalServerError
		}
		args = append(args,
			"--allowed-languages", identifier.LanguageC,
			"--allowed-languages", identifier.LanguageJava,
			"--patch-url", patchURL,
		)
		cacheToHash = append(cacheToHash, "patch", *patchBlob)
	}

	if h.JobClient == nil {
		span.SetStatus(codes.Error, "jobClient was nil")
		span.RecordError(errors.New("job client was nil"))
		return nil, response.InternalServerError
	}

	cacheKey := sha256.Sum256([]byte(strings.Join(cacheToHash, "\n")))

	job := &models.Job{}
	err := db.Transaction(func(db *gorm.DB) error {
		span.AddEvent("creating if not exists")
		var result *gorm.DB
		if *jobRequest.OverrideCache {
			job.CacheKey = hex.EncodeToString(cacheKey[:])
			result = db.Create(job)
		} else {
			result = db.Model(job).Order("created_at desc").
				FirstOrCreate(job, models.Job{CacheKey: hex.EncodeToString(cacheKey[:])})
		}
		if result.Error != nil {
			return result.Error
		}
		jobID := job.ID.String()
		span.SetAttributes(attribute.String("job.id", jobID))
		args = append(args, "--job-id", jobID)

		if result.RowsAffected == 1 {
			span.AddEvent("starting job")
			_, err := h.JobClient.CreateEvalJob(
				ctx,
				types.JobTypeJob,
				jobID,
				args,
				task.MemoryGB,
				task.CPUs,
				nil,
				nil,
				nil,
			)
			if err != nil {
				return err
			}
		} else {
			span.AddEvent("cache_hit")
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to initialize job")
		return nil, response.InternalServerError
	}

	presigned, err := h.presignJob(ctx, job)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get presigned")
		return nil, response.InternalServerError
	}

	return &types.JobResponse{
		JobID:                     job.ID.String(),
		Status:                    job.Status,
		FunctionalityTestsPassing: models.PtrFromNull(job.FunctionalityTestsPassing),
		Results:                   presigned.Results,
		Artifacts:                 presigned.Artifacts,
	}, nil
}
