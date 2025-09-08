package v1

import (
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
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) SubmitSarifAssessment(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitSarifAssessment")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received sarif assessment submission request")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	task, ok := c.Get("task").(*models.Task)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	sarif, ok := c.Get("sarif").(*models.SARIFBroadcast)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("sarif: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	teamID := auth.ID.String()
	taskID := task.ID.String()
	broadcastSARIFID := sarif.ID.String()
	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.String("broadcastSARIF.id", broadcastSARIFID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		types.SarifAssessmentSubmission
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

	span.AddEvent("checking that broadcast SARIF task ID matches task ID")
	if sarif.TaskID != task.ID {
		span.SetStatus(codes.Error, "task ID does not match broadcast SARIF ID")
		span.RecordError(err)
		return response.NotFoundError
	}

	sarifAssessment := models.SARIFAssessment{
		SubmitterID:      auth.ID,
		SARIFBroadcastID: sarif.ID,
		Assessment:       rdata.Assessment,
		Status:           types.SubmissionStatusAccepted,
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		sarifAssessment.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(sarifAssessment.Status)),
		))
	}

	span.AddEvent("inserting into database")
	err = db.Create(&sarifAssessment).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert SARIF assessment")
		span.RecordError(err)
		return response.InternalServerError
	}
	sarifAssessmentID := sarifAssessment.ID.String()
	span.SetAttributes(attribute.String("sarifAssessment.id", sarifAssessmentID))

	span.AddEvent("generating audit log message")
	audit.LogSARIFAssessment(audit.Context{
		RoundID: task.RoundID,
		TaskID:  &taskID,
		TeamID:  &teamID,
	}, sarifAssessmentID, string(rdata.Assessment), broadcastSARIFID)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "Success")
	return c.JSON(http.StatusOK, types.SarifAssessmentResponse{Status: sarifAssessment.Status})
}
