package v1

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/archive"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func (h *Handler) SubmitFreeform(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitFreeform")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received freeform submission request")

	task, ok := c.Get("task").(*models.Task)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	taskID := task.ID.String()
	teamID := auth.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		types.FreeformSubmission
	}
	var rdata requestData

	span.AddEvent("parsing request body")
	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to validate request data")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	span.AddEvent("validating submission is within size limit")
	if !validator.ValidateFreeformSize(len(rdata.Submission)) {
		span.SetStatus(codes.Ok, "submission was too large")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"submission": "must be <= 2mb",
			}},
		)
	}

	span.AddEvent("decoding submission base64")
	submission, err := base64.StdEncoding.DecodeString(rdata.Submission)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to decode submission")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"submission": "must be valid base64",
			}},
		)
	}

	freeform := &models.FreeformSubmission{
		TaskID:      task.ID,
		SubmitterID: auth.ID,
		Submission:  string(submission),
		Status:      types.SubmissionStatusAccepted,
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		freeform.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(freeform.Status)),
		))
	}

	span.AddEvent("inserting into database")
	err = db.Create(freeform).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert")
		span.RecordError(err)
		return response.InternalServerError
	}
	freeformID := freeform.ID.String()
	span.SetAttributes(attribute.String("freeform.id", freeformID))

	auditContext := audit.Context{RoundID: task.RoundID, TaskID: &taskID, TeamID: &teamID}
	submissionBytes := []byte(rdata.Submission)
	upload := &archive.FileMetadata{
		Buffer:       &submissionBytes,
		ArchivedFile: types.FileFreeformPOV,
		Entity:       audit.EntityFreeformPOV,
		EntityID:     freeformID,
	}

	err = archive.ArchiveFile(ctx, auditContext, h.archiver, upload)
	if err != nil {
		span.SetStatus(codes.Error, "failed to archive file")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.AddEvent("generating audit log message")
	audit.LogFreeformSubmission(auditContext)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")

	return c.JSON(http.StatusOK, types.FreeformResponse{
		FreeformID: freeform.ID.String(),
		Status:     freeform.Status})
}
