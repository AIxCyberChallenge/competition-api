package workerqueue

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/worker/internal/workerqueue",
)

type WorkerQueuer struct {
	queuer     queue.Queuer
	entityID   string
	entityType types.JobType
}

func NewWorkerQueue(entityID string, entityType types.JobType, queuer queue.Queuer) *WorkerQueuer {
	return &WorkerQueuer{
		entityID:   entityID,
		entityType: entityType,
		queuer:     queuer,
	}
}

func (q *WorkerQueuer) Artifact(ctx context.Context, artifact types.JobArtifact) error {
	ctx, span := tracer.Start(ctx, "WorkerQueuer.Artifact", trace.WithAttributes(
		attribute.String("entity.type", string(q.entityType)),
		attribute.String("entity.id", q.entityID),
	))
	defer span.End()

	err := q.queuer.Enqueue(
		ctx,
		types.NewWorkerMsgArtifact(q.entityType, q.entityID, artifact),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "enqueued message")
	return nil
}

func (q WorkerQueuer) CommandResult(ctx context.Context, result *types.JobResult) error {
	ctx, span := tracer.Start(ctx, "WorkerQueuer.CommandResult", trace.WithAttributes(
		attribute.String("entity.type", string(q.entityType)),
		attribute.String("entity.id", q.entityID),
	))
	defer span.End()

	err := q.queuer.Enqueue(ctx, types.NewWorkerMsgCommandResult(q.entityType, q.entityID, result))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "enqueued message")
	return nil
}

func (q WorkerQueuer) FinalMessage(
	ctx context.Context,
	status types.SubmissionStatus,
	patchTestsFailed bool,
) error {
	ctx, span := tracer.Start(ctx, "WorkerQueuer.FinalMessage", trace.WithAttributes(
		attribute.String("entity.type", string(q.entityType)),
		attribute.String("entity.id", q.entityID),
	))
	defer span.End()

	err := q.queuer.Enqueue(
		ctx,
		types.NewWorkerMsgFinal(q.entityType, q.entityID, status, &patchTestsFailed),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "enqueued message")
	return nil
}
