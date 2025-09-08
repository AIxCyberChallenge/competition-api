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

func (h *Handler) SubmitBundle(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitBundle")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received bundle submission request")

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

	teamID := auth.ID.String()
	taskID := task.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		types.BundleSubmission
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

	span.AddEvent("validating bundle contains enough fields to be meaningful")
	err = rdata.ValidateFieldCount()
	if err != nil {
		span.SetStatus(codes.Ok, "bundle does not have enough fields set")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("must set at least 2 fields"),
		)
	}

	span.AddEvent("parsing fields")
	parsed, err := rdata.Parse()
	if err != nil {
		span.SetStatus(codes.Ok, "failed to parse fields")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError("failed to parse fields"))
	}

	bundle := &models.Bundle{
		SubmitterID: auth.ID,
		TaskID:      task.ID,
		Status:      types.SubmissionStatusAccepted,
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		bundle.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(bundle.Status)),
		))
	}

	span.AddEvent("checking IDs in bundle are valid")
	valid, err := bundle.CheckAndSetRelations(db, parsed)
	if err != nil {
		span.SetStatus(codes.Error, "failed to check relations")
		span.RecordError(err)
		return response.InternalServerError
	}

	if !valid {
		span.SetStatus(codes.Ok, "bundle IDs are invalid")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.AddEvent("inserting into database")
	err = db.Create(bundle).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert")
		span.RecordError(err)
		return response.InternalServerError
	}
	bundleID := bundle.ID.String()
	span.SetAttributes(attribute.String("bundle.id", bundleID))

	span.AddEvent("generating audit log message")
	auditContext := audit.Context{RoundID: task.RoundID, TeamID: &teamID, TaskID: &taskID}
	audit.LogBundleSubmission(auditContext,
		bundleID,
		models.PtrFromNull(bundle.POVID),
		models.PtrFromNull(bundle.PatchID),
		models.PtrFromNull(bundle.BroadcastSARIFID),
		models.PtrFromNull(bundle.SubmittedSARIFID),
		models.PtrFromNull(bundle.Description),
		models.PtrFromNull(bundle.FreeformID),
		bundle.Status,
	)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(
		http.StatusOK,
		types.BundleSubmissionResponse{BundleID: bundleID, Status: bundle.Status},
	)
}

func (h *Handler) PatchBundle(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "PatchBundle")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received bundle patch request")

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

	bundle, ok := c.Get("bundle").(*models.Bundle)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("bundle: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	teamID := auth.ID.String()
	taskID := task.ID.String()
	bundleID := bundle.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.String("bundle.id", bundleID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		types.BundleSubmission
	}
	var rdata requestData

	span.AddEvent("checking user can perform this operation")
	if bundle.SubmitterID != auth.ID || bundle.TaskID != task.ID {
		span.SetStatus(codes.Ok, "submitter or task did not match")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(bundle.Status)),
		))
		span.SetStatus(codes.Ok, "")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("deadline to modify submission passed"),
		)
	}

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

	span.AddEvent("parsing fields")
	parsed, err := rdata.Parse()
	if err != nil {
		span.SetStatus(codes.Ok, "failed to parse fields")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError("failed to parse fields"))
	}

	span.AddEvent("validating modified bundle contains enough fields to be meaningful")
	countSet := bundle.CountNotNull(parsed)
	if countSet < 2 {
		span.SetStatus(codes.Ok, "bundle does not have enough fields set")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("must have at least 2 fields set after update"),
		)
	}

	span.AddEvent("checking IDs in bundle are valid")
	valid, err := bundle.CheckAndSetRelations(db, parsed)
	if err != nil {
		span.SetStatus(codes.Error, "failed to check relations")
		span.RecordError(err)
		return response.InternalServerError
	}

	if !valid {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "bundle IDs are invalid")
		return response.NotFoundError
	}

	span.AddEvent("updating bundle in database")
	err = db.Save(bundle).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to update bundle")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.AddEvent("generating audit log message")
	auditContext := audit.Context{RoundID: task.RoundID, TeamID: &teamID, TaskID: &taskID}
	audit.LogBundleSubmission(auditContext,
		bundleID,
		models.PtrFromNull(bundle.POVID),
		models.PtrFromNull(bundle.PatchID),
		models.PtrFromNull(bundle.BroadcastSARIFID),
		models.PtrFromNull(bundle.SubmittedSARIFID),
		models.PtrFromNull(bundle.Description),
		models.PtrFromNull(bundle.FreeformID),
		bundle.Status,
	)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(http.StatusOK, types.BundleSubmissionResponseVerbose{
		BundleSubmissionResponse: types.BundleSubmissionResponse{
			BundleID: bundleID,
			Status:   bundle.Status,
		},
		BundleSubmissionResponseBody: types.BundleSubmissionResponseBody{
			POVID:            models.PtrFromNull(bundle.POVID),
			PatchID:          models.PtrFromNull(bundle.PatchID),
			BroadcastSARIFID: models.PtrFromNull(bundle.BroadcastSARIFID),
			SubmittedSARIFID: models.PtrFromNull(bundle.SubmittedSARIFID),
			Description:      models.PtrFromNull(bundle.Description),
			FreeformID:       models.PtrFromNull(bundle.FreeformID),
		},
	})
}

func (*Handler) GetBundle(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "GetBundle")
	defer span.End()

	span.AddEvent("received bundle get request")

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

	bundle, ok := c.Get("bundle").(*models.Bundle)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("bundle: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	teamID := auth.ID.String()
	taskID := task.ID.String()
	bundleID := bundle.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.String("bundle.id", bundleID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	span.AddEvent("checking user can perform this operation")
	if bundle.SubmitterID != auth.ID || bundle.TaskID != task.ID {
		span.SetStatus(codes.Ok, "submitter or task did not match")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(http.StatusOK, types.BundleSubmissionResponseVerbose{
		BundleSubmissionResponse: types.BundleSubmissionResponse{
			BundleID: bundleID,
			Status:   bundle.Status,
		},
		BundleSubmissionResponseBody: types.BundleSubmissionResponseBody{
			POVID:            models.PtrFromNull(bundle.POVID),
			PatchID:          models.PtrFromNull(bundle.PatchID),
			BroadcastSARIFID: models.PtrFromNull(bundle.BroadcastSARIFID),
			SubmittedSARIFID: models.PtrFromNull(bundle.SubmittedSARIFID),
			Description:      models.PtrFromNull(bundle.Description),
			FreeformID:       models.PtrFromNull(bundle.FreeformID),
		},
	})
}

func (h *Handler) DeleteBundle(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "DeleteBundle")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received bundle delete request")

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
	bundle, ok := c.Get("bundle").(*models.Bundle)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("bundle: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	teamID := auth.ID.String()
	taskID := task.ID.String()
	bundleID := bundle.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", task.RoundID),
		attribute.String("task.id", taskID),
		attribute.String("team.id", teamID),
		attribute.String("bundle.id", bundleID),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	span.AddEvent("checking user can perform this operation")
	if bundle.SubmitterID != auth.ID || bundle.TaskID != task.ID {
		span.SetStatus(codes.Ok, "submitter or task did not match")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.AddEvent("checking deadline has not passed")
	if requestTime.After(task.Deadline) {
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(bundle.Status)),
		))
		span.SetStatus(codes.Ok, "")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("deadline to modify submission passed"),
		)
	}

	span.AddEvent("deleting bundle from database")
	err := db.Delete(bundle).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to delete bundle")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.AddEvent("generating audit log message")
	auditContext := audit.Context{RoundID: task.RoundID, TeamID: &teamID, TaskID: &taskID}
	audit.LogBundleDelete(auditContext, bundleID)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.NoContent(http.StatusNoContent)
}
