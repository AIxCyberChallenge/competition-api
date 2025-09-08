package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

type WorkerMsgHandler struct {
	db              *gorm.DB
	archiver        upload.Uploader
	artifactFetcher fetch.Fetcher
}

var _ queue.MessageHandler = (*WorkerMsgHandler)(nil)

func (h *WorkerMsgHandler) HandleArtifactMessage(
	ctx context.Context,
	msg *types.WorkerMsgArtifact,
) error {
	ctx, span := tracer.Start(ctx, "HandleArtifactMessage")
	defer span.End()

	db := h.db.WithContext(ctx)

	switch msg.Entity {
	case types.JobTypeJob:
		metaJSON, err := json.Marshal([]types.JobArtifact{msg.Artifact})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to encode artifact metadata as JSON")
			return err
		}

		err = db.Model(&models.Job{}).Where("id = ?", msg.EntityID).Update(
			"artifacts", gorm.Expr("CASE WHEN artifacts IS NULL THEN '[]'::jsonb ELSE artifacts END || ?::jsonb", metaJSON),
		).
			Error
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to update job in DB with artifact")
			return err
		}
	case types.JobTypePOV:
		fallthrough
	case types.JobTypePatch:
		body, err := h.artifactFetcher.Fetch(ctx, msg.Artifact.Blob.ObjectName)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch object")
			return err
		}
		defer body.Close()

		// FIXME: send to a temp file
		buffer := bytes.Buffer{}
		_, err = io.Copy(&buffer, body)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to download object to memory")
			return err
		}

		storeIdentifier, err := h.archiver.StoreIdentifier(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get store identifier")
		}

		err = h.archiver.Upload(
			ctx,
			bytes.NewReader(buffer.Bytes()),
			int64(buffer.Len()),
			msg.Artifact.Blob.ObjectName,
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to upload file to s3")
			return err
		}

		var row models.Submission
		entity := audit.FileArchivedEntity(msg.Entity)

		switch entity {
		case audit.EntityPOV:
			row, err = models.ByID[models.POVSubmission](ctx, db, uuid.MustParse(msg.EntityID))
		case audit.EntityPatch:
			row, err = models.ByID[models.PatchSubmission](ctx, db, uuid.MustParse(msg.EntityID))
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get entity for artifact")
			return err
		}

		teamID := row.GetSubmitterID().String()
		taskID := row.GetTaskID().String()
		roundID, err := models.GetRoundIDForSubmission(ctx, db, row)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get round id for artifact")
			return err
		}

		audit.LogFileArchived(
			audit.Context{RoundID: roundID, TeamID: &teamID, TaskID: &taskID},
			storeIdentifier,
			msg.Artifact.Blob.ObjectName,
			msg.Artifact.ArchivedFile,
			audit.FileArchivedEntity(msg.Entity),
			msg.EntityID)
	}

	return nil
}

func (h *WorkerMsgHandler) HandleCommandResultMessage(
	ctx context.Context,
	msg *types.WorkerMsgCommandResult,
) error {
	_, span := tracer.Start(ctx, "HandleCommandResultMessage")
	defer span.End()

	db := h.db.WithContext(ctx)

	if msg.Entity != types.JobTypeJob {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "skipped handling message because not a job")
		return nil
	}

	if msg.Result == nil {
		err := errors.New("empty command result message")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	cmdResultJSON, err := json.Marshal(msg.Result)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to encode command result as JSON")
		return err
	}

	err = db.Model(&models.Job{}).Where("id = ?", msg.EntityID).Update(
		"results", gorm.Expr("CASE WHEN results IS NULL THEN '[]'::jsonb ELSE results END || ?::jsonb", cmdResultJSON),
	).
		Error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update job in DB with command result")
		return err
	}
	return nil
}

func (h *WorkerMsgHandler) HandleFinalMessage(
	ctx context.Context,
	msg *types.WorkerMsgFinal,
) error {
	_, span := tracer.Start(ctx, "HandleFinalMessage", trace.WithAttributes(
		attribute.String("msg.status", string(msg.Status)),
		attribute.Bool("msg.patchTestsFailed", msg.PatchTestsFailed),
	))
	defer span.End()

	db := h.db.WithContext(ctx)

	entityUUID, err := uuid.Parse(msg.EntityID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse entity ID as UUID")
		return queue.WrapPoisonError(fmt.Errorf("failed to parse entity ID as UUID: %w", err))
	}
	err = db.Transaction(func(db *gorm.DB) error {
		var result *gorm.DB
		var submission models.Submission
		switch msg.Entity {
		case types.JobTypePOV:
			submission = &models.POVSubmission{}
			result = db.Model(submission).
				Clauses(clause.Returning{}).
				Where("id = ?", entityUUID).
				Where("status = ?", types.SubmissionStatusAccepted).
				Updates(
					models.POVSubmission{
						Status: msg.Status,
					},
				)
		case types.JobTypePatch:
			testsFailed := false
			if msg.Status == types.SubmissionStatusFailed {
				testsFailed = msg.PatchTestsFailed
			}

			submission = &models.PatchSubmission{}
			result = db.Model(submission).
				Clauses(clause.Returning{}).
				Where("id = ?", entityUUID).
				Where("status = ?", types.SubmissionStatusAccepted).
				Updates(
					models.PatchSubmission{
						Status:                    msg.Status,
						FunctionalityTestsPassing: models.NewNullFromData(!testsFailed),
					},
				)
		case types.JobTypeJob:
			testsFailed := false
			if msg.Status == types.SubmissionStatusFailed {
				testsFailed = msg.PatchTestsFailed
			}
			result = db.Model(&models.Job{}).
				Clauses(clause.Returning{}).
				Where("id = ?", entityUUID).
				Where("status = ?", types.SubmissionStatusAccepted).
				Updates(
					models.Job{
						Status:                    msg.Status,
						FunctionalityTestsPassing: models.NewNullFromData(!testsFailed),
					},
				)
		default:
			err = fmt.Errorf("unsupported entity: %s", msg.Entity)
			span.RecordError(err)
			span.SetStatus(codes.Error, "unsupported entity")
			return queue.WrapPoisonError(err)
		}

		if result.Error != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return nil
		}

		if submission != nil {
			teamID := submission.GetSubmitterID().String()
			taskID := submission.GetTaskID().String()
			roundID, err := models.GetRoundIDForSubmission(ctx, db, submission)
			if err != nil {
				return err
			}
			c := audit.Context{TeamID: &teamID, TaskID: &taskID, RoundID: roundID}
			submission.AuditLogSubmissionResult(c)
		}

		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update the db state")
		return err
	}

	return nil
}

func (h *WorkerMsgHandler) Handle(
	ctx context.Context,
	message []byte,
) error {
	ctx, span := tracer.Start(ctx, "WorkerMsgHandler.HandleMessage", trace.WithNewRoot())
	defer span.End()

	var baseMsg types.WorkerMsg
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to unmarshal queue message into generic type")
		return queue.WrapPoisonError(err)
	}

	span.SetAttributes(
		attribute.String("msg.type", string(baseMsg.MsgType)),
		attribute.String("msg.entity.type", string(baseMsg.Entity)),
		attribute.String("msg.entity.id", baseMsg.EntityID),
	)
	switch baseMsg.MsgType {
	case types.MsgTypeArtifact:
		specMsg := types.WorkerMsgArtifact{}
		if err := json.Unmarshal(message, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal queue message into specific type")
			return queue.WrapPoisonError(err)
		}

		if err := h.HandleArtifactMessage(ctx, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to handle")
			return err
		}
	case types.MsgTypeCommandResult:
		specMsg := types.WorkerMsgCommandResult{}
		if err := json.Unmarshal(message, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal queue message into specific type")
			return queue.WrapPoisonError(err)
		}

		if err := h.HandleCommandResultMessage(ctx, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to handle")
			return err
		}

	case types.MsgTypeFinal:
		specMsg := types.WorkerMsgFinal{}
		if err := json.Unmarshal(message, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal queue message into specific type")
			return queue.WrapPoisonError(err)
		}

		if err := h.HandleFinalMessage(ctx, &specMsg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to handle")
			return err
		}
	default:
		err := errors.New("queue message type not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return queue.WrapPoisonError(err)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "handled")
	return nil
}

// Monitors queue results and handles them until `ctx` is cancelled
func MonitorResultsQueue(
	ctx context.Context,
	db *gorm.DB,
	qr queue.Queuer,
	archiver upload.Uploader,
	artifactFetcher fetch.Fetcher,
) {
	ctx, span := tracer.Start(ctx, "MonitorResultsQueue")
	defer span.End()
	handler := &WorkerMsgHandler{
		db:              db,
		archiver:        archiver,
		artifactFetcher: artifactFetcher,
	}
OUTER:
	for {
		func() {
			//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
			ctx, span := tracer.Start(ctx, "MonitorResultsQueue.Loop")
			defer span.End()

			if err := qr.Dequeue(ctx, 10*time.Minute, handler); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to dequeue and handle message")
				return
			}
		}()

		select {
		case <-ctx.Done():
			break OUTER
		default:
			continue
		}
	}
}
