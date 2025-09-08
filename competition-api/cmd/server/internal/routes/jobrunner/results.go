package jobrunner

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/codes"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) PostJobResultsBulk(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "PostJobResultsBulk")
	defer span.End()

	db := h.DB.WithContext(ctx)

	type requestData struct {
		Jobs []uuid.UUID `json:"jobs" validate:"required"`
	}

	var rdata requestData

	span.AddEvent("parsing request body")
	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to validate request data")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	if len(rdata.Jobs) > 100 || len(rdata.Jobs) < 1 {
		span.SetStatus(codes.Error, "invalid job count")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	var jobs []models.Job
	result := db.Model(models.Job{}).Where("id in ?", rdata.Jobs).Order("id desc").Find(&jobs)
	if result.Error != nil {
		span.SetStatus(codes.Error, "failed to query jobs")
		span.RecordError(result.Error)
		return response.InternalServerError
	}

	jobResponses := make([]types.JobResponse, 0, len(jobs))
	for _, job := range jobs {
		presigned, err := h.presignJob(ctx, &job)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to presign")
			return response.InternalServerError
		}
		jobResponses = append(jobResponses, types.JobResponse{
			JobID:                     job.ID.String(),
			Status:                    job.Status,
			FunctionalityTestsPassing: models.PtrFromNull(job.FunctionalityTestsPassing),
			Artifacts:                 presigned.Artifacts,
			Results:                   presigned.Results,
		})
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "get job results")
	return c.JSON(http.StatusOK, jobResponses)
}

func (h *Handler) GetJobResults(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "GetJobResults")
	defer span.End()

	job, ok := c.Get("job").(*models.Job)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("job: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	presigned, err := h.presignJob(ctx, job)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to presign")
		return response.InternalServerError
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "get job results")
	return c.JSON(http.StatusOK, types.JobResponse{
		JobID:                     job.ID.String(),
		Status:                    job.Status,
		FunctionalityTestsPassing: models.PtrFromNull(job.FunctionalityTestsPassing),
		Artifacts:                 presigned.Artifacts,
		Results:                   presigned.Results,
	})
}
