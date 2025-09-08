package v1

import (
	"bytes"
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
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/identifier"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func (h *Handler) SubmitPatch(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitPatch")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received patch submission request")

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

	span.AddEvent("validating that we can test this patch")
	if !task.HarnessesIncluded {
		span.SetStatus(codes.Ok, "can't test a patch against a task with no harnesses")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError(
				"cannot submit patch against task with no harnesses. use freeform patch endpoint instead",
			),
		)
	}

	type requestData struct {
		types.PatchSubmission
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
	if !validator.ValidatePatchSize(len(rdata.Patch)) {
		span.SetStatus(codes.Ok, "submission was too large")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"patch": "must be <= 100kb",
			}},
		)
	}

	span.AddEvent("decoding submission base64")
	patchData, err := base64.StdEncoding.DecodeString(rdata.Patch)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to decode submission")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"patch": "must be valid base64",
			}},
		)
	}

	span.AddEvent("uploading submission")
	blobName, err := upload.Hashed(
		ctx,
		h.submissionUploader,
		bytes.NewReader(patchData),
		int64(len(patchData)),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to upload submission")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("patch.hash", blobName),
		attribute.String("blob.name", blobName),
	)

	patch := models.PatchSubmission{
		SubmitterID:   auth.ID,
		PatchFilePath: blobName,
		Status:        types.SubmissionStatusAccepted,
		TaskID:        task.ID,
	}

	deadlinePassed := requestTime.After(task.Deadline)
	span.AddEvent("checking deadline has not passed")
	if deadlinePassed {
		patch.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(patch.Status)),
		))
	}

	span.AddEvent("inserting into database")
	err = db.Create(&patch).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert")
		span.RecordError(err)
		return response.InternalServerError
	}
	patchID := patch.ID.String()
	span.SetAttributes(attribute.String("patch.id", patchID))

	auditContext := audit.Context{RoundID: task.RoundID, TaskID: &taskID, TeamID: &teamID}

	metadata := &archive.FileMetadata{
		Buffer:       &patchData,
		ArchivedFile: types.FilePatch,
		Entity:       audit.EntityPatch,
		EntityID:     patchID,
	}
	if err := archive.ArchiveFile(ctx, auditContext, h.archiver, metadata); err != nil {
		span.SetStatus(codes.Error, "failed to archive file")
		span.RecordError(err)
		return response.InternalServerError
	}

	if !deadlinePassed && h.JobClient != nil {
		expiration := time.Hour * 100

		span.AddEvent("getting presigned submission URL for job kickoff", trace.WithAttributes(
			attribute.String("expiration", expiration.String()),
		))
		patchURL, err := h.submissionUploader.PresignedReadURL(ctx, patch.PatchFilePath, expiration)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned url")
			span.RecordError(err)
			return response.InternalServerError
		}

		span.AddEvent("getting presigned source URLs for job kickoff", trace.WithAttributes(
			attribute.String("expiration", expiration.String()),
		))
		sources, err := task.GetSourceURLs(ctx, h.sourcesUploader, expiration)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get presigned url")
			span.RecordError(err)
			return response.InternalServerError
		}

		args := []string{
			"--head-repo-url", sources.HeadRepo,
			"--focus", task.Focus,
			"--project-name", task.ProjectName,
			"--oss-fuzz-url", sources.FuzzTooling,
			"--architecture", "x86_64",
			"--patch-url", patchURL,
			"--allowed-languages", identifier.LanguageC,
			"--allowed-languages", identifier.LanguageJava,
			"--patch-id", patch.ID.String(),
		}

		taskID := task.ID.String()
		authID := auth.ID.String()
		span.AddEvent("starting job")
		_, err = h.JobClient.CreateEvalJob(
			ctx,
			types.JobTypePatch,
			patch.ID.String(),
			args,
			task.MemoryGB,
			task.CPUs,
			&task.RoundID,
			&taskID,
			&authID,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to start job")
			span.RecordError(err)
			return response.InternalServerError
		}
	}

	span.AddEvent("generating audit log message")
	audit.LogPatchSubmission(auditContext, patchID, patch.Status, blobName)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")

	// TODO: set functionality pass even if it is impossible to be true here
	return c.JSON(http.StatusOK, types.PatchSubmissionResponse{
		PatchID:                   patch.ID.String(),
		Status:                    patch.Status,
		FunctionalityTestsPassing: models.PtrFromNull(patch.FunctionalityTestsPassing),
	})
}

func (*Handler) PatchStatus(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "PatchStatus")
	defer span.End()
	span.AddEvent("received patch status request")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	patch, ok := c.Get("patch").(*models.PatchSubmission)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("patch: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	task, ok := c.Get("task").(*models.Task)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.AddEvent("checking user can perform this operation")
	if patch.SubmitterID != auth.ID || task.ID != patch.TaskID {
		span.SetStatus(codes.Ok, "submitter or task did not match")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")

	return c.JSON(http.StatusOK, types.PatchSubmissionResponse{
		PatchID:                   patch.ID.String(),
		Status:                    patch.Status,
		FunctionalityTestsPassing: models.PtrFromNull(patch.FunctionalityTestsPassing),
	})
}
