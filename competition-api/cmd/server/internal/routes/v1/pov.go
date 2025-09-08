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
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func (h *Handler) SubmitPOV(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "SubmitPOV")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.AddEvent("received pov submission request")

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

	span.AddEvent("validating that we can test this pov")
	if !task.HarnessesIncluded {
		span.SetStatus(codes.Error, "can't test a pov with no harnesses")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError(
				"cannot submit pov for vuln with no harnesses. use freeform pov instead",
			),
		)
	}

	type requestData struct {
		types.POVSubmission
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
	if !validator.ValidateTriggerSize(len(rdata.Testcase)) {
		span.SetStatus(codes.Ok, "submission was too large")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"testcase": "must be <= 2mb",
			}},
		)
	}

	span.AddEvent("decoding submission base64")
	vulnData, err := base64.StdEncoding.DecodeString(rdata.Testcase)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to decode submission")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"testcase": "must be valid base64",
			}},
		)
	}

	span.AddEvent("uploading submission")
	blobName, err := upload.Hashed(
		ctx,
		h.submissionUploader,
		bytes.NewReader(vulnData),
		int64(len(vulnData)),
	)
	if err != nil {
		span.SetStatus(codes.Error, "failed to upload submission")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("testcase.hash", blobName),
		attribute.String("blob.name", blobName),
	)

	povSubmission := models.POVSubmission{
		SubmitterID:  auth.ID,
		TaskID:       task.ID,
		TestcasePath: blobName,
		FuzzerName:   rdata.FuzzerName,
		Sanitizer:    rdata.Sanitizer,
		Architecture: string(rdata.Architecture),
		Status:       types.SubmissionStatusAccepted,
		Engine:       string(rdata.Engine),
	}

	deadlinePassed := requestTime.After(task.Deadline)
	span.AddEvent("checking deadline has not passed")
	if deadlinePassed {
		povSubmission.Status = types.SubmissionStatusDeadlineExceeded
		span.AddEvent("deadline exceeded", trace.WithAttributes(
			attribute.Int64("deadline_ms", task.Deadline.UnixMilli()),
			attribute.String("status", string(povSubmission.Status)),
		))
	}

	span.AddEvent("inserting into database")
	err = db.Create(&povSubmission).Error
	if err != nil {
		span.SetStatus(codes.Error, "failed to insert")
		span.RecordError(err)
		return response.InternalServerError
	}
	povID := povSubmission.ID.String()
	span.SetAttributes(attribute.String("pov.id", povID))

	auditContext := audit.Context{RoundID: task.RoundID, TaskID: &taskID, TeamID: &teamID}
	metadata := &archive.FileMetadata{
		Buffer:       &vulnData,
		ArchivedFile: types.FilePOVTrigger,
		Entity:       audit.EntityPOV,
		EntityID:     povID,
	}
	err = archive.ArchiveFile(ctx, auditContext, h.archiver, metadata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to archive file")
		span.RecordError(err)
		return response.InternalServerError
	}

	if !deadlinePassed && h.JobClient != nil {
		expiration := time.Hour * 100

		span.AddEvent("getting presigned submission URL for job kickoff", trace.WithAttributes(
			attribute.String("expiration", expiration.String()),
		))
		triggerURL, err := h.submissionUploader.PresignedReadURL(
			ctx,
			povSubmission.TestcasePath,
			expiration,
		)
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
			"--architecture", povSubmission.Architecture,
			"--trigger-url", triggerURL,
			"--sanitizer", povSubmission.Sanitizer,
			"--harness-name", povSubmission.FuzzerName,
			"--engine", povSubmission.Engine,
			"--pov-id", povSubmission.ID.String(),
			"--archive-s3",
		}

		// this is only set if the task type is delta and if it exists in the getSourceURLS function
		if sources.BaseRepo != "" {
			args = append(args, "--base-repo-url", sources.BaseRepo)
		}

		taskID := task.ID.String()
		authID := auth.ID.String()
		span.AddEvent("starting job")
		_, err = h.JobClient.CreateEvalJob(
			ctx,
			types.JobTypePOV,
			povSubmission.ID.String(),
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
	audit.LogPOVSubmission(
		auditContext,
		povSubmission.ID.String(),
		povSubmission.Status,
		rdata.FuzzerName,
		blobName,
		rdata.Sanitizer,
		string(rdata.Architecture),
		povSubmission.Engine,
	)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")

	return c.JSON(http.StatusOK, types.POVSubmissionResponse{
		POVID:  povSubmission.ID.String(),
		Status: povSubmission.Status,
	})
}

func (*Handler) POVStatus(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "POVStatus")
	defer span.End()
	span.AddEvent("received pov status request")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	vuln, ok := c.Get("pov").(*models.POVSubmission)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("pov: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	task, ok := c.Get("task").(*models.Task)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.AddEvent("checking user can perform this operation")
	if vuln.SubmitterID != auth.ID ||
		vuln.TaskID != task.ID {
		span.SetStatus(codes.Ok, "vuln submitter ID did not match auth ID")
		span.RecordError(nil)
		return response.NotFoundError
	}

	span.SetStatus(codes.Ok, "")
	span.RecordError(nil)
	return c.JSON(http.StatusOK, types.POVSubmissionResponse{
		POVID:  vuln.ID.String(),
		Status: vuln.Status,
	})
}
