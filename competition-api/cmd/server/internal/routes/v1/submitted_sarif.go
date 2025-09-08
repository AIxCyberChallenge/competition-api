package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/archive"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/sarif"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) SubmitSarif(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitSarif")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received sarif submission request")

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
		types.SARIFSubmission
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

	span.AddEvent("validating submitted SARIF against schema")
	err = sarif.Schema.Validate(*rdata.SARIF)
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		errs := validationErr.BasicOutput().Errors
		fieldMap := make(map[string]string, len(errs))
		for _, err := range errs {
			fieldMap[err.KeywordLocation] = err.Error
		}
		span.SetStatus(codes.Ok, "submitted SARIF was not schema compliant")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "sarif failed to validate", Fields: &fieldMap},
		)
	} else if err != nil {
		span.SetStatus(codes.Ok, "failed to validate submitted SARIF")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError(err.Error()))
	}

	span.AddEvent("encoding submitted SARIF as JSON")
	rawSARIF, err := json.Marshal(*rdata.SARIF)
	if err != nil {
		span.SetStatus(codes.Error, "failed to encode submitted SARIF")
		span.RecordError(err)
		return response.InternalServerError
	}

	sarifSubmission := &models.SARIFSubmission{
		SubmitterID: auth.ID,
		TaskID:      task.ID,
		SARIF:       datatypes.JSON(rawSARIF),
		Status:      types.SubmissionStatusAccepted,
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		sarifSubmission.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(sarifSubmission.Status)),
		))
	}

	span.AddEvent("inserting into database")
	err = db.Create(sarifSubmission).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert")
		span.RecordError(err)
		return response.InternalServerError
	}
	submittedSARIFID := sarifSubmission.ID.String()
	span.SetAttributes(attribute.String("submittedSARIF.id", submittedSARIFID))

	auditContext := audit.Context{RoundID: task.RoundID, TaskID: &taskID, TeamID: &teamID}
	upload := &archive.FileMetadata{
		Buffer:       &rawSARIF,
		ArchivedFile: types.FileSARIFSubmission,
		Entity:       audit.EntitySARIFSubmission,
		EntityID:     submittedSARIFID,
	}

	err = archive.ArchiveFile(ctx, auditContext, h.archiver, upload)
	if err != nil {
		span.SetStatus(codes.Error, "failed to archive file")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.AddEvent("generating audit log message")
	audit.LogSARIFSubmission(auditContext, submittedSARIFID, sarifSubmission.Status)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(http.StatusOK, types.SARIFSubmissionResponse{
		SubmittedSARIFID: sarifSubmission.ID.String(),
		Status:           sarifSubmission.Status,
	})
}
