package engine

import (
	"context"
	"errors"
	"time"

	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/otel/codes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

// Ensure BuildRetryEngine implementes Engine interface
var _ Engine = (*BuildRetryEngine)(nil)

type BuildRetryEngine struct {
	factory func() retry.Backoff
	engine  Engine
}

func NewBuildRetryEngine(engine Engine) *BuildRetryEngine {
	return &BuildRetryEngine{
		engine: engine,
		factory: func() retry.Backoff {
			b := retry.NewConstant(time.Second * 30)
			b = retry.WithMaxRetries(3, b)
			return b
		},
	}
}

func NewBuildRetryEngineWithFactory(engine Engine, factory func() retry.Backoff) *BuildRetryEngine {
	return &BuildRetryEngine{
		engine:  engine,
		factory: factory,
	}
}

// ApplyPatch implements Engine.
func (r *BuildRetryEngine) ApplyPatch(ctx context.Context, data *Params, patchPath string) error {
	return r.engine.ApplyPatch(ctx, data, patchPath)
}

// Build implements Engine.
func (r *BuildRetryEngine) Build(ctx context.Context, data *Params) error {
	ctx, span := tracer.Start(ctx, "RetryEngine.Build")
	defer span.End()

	errs := []error{}
	err := retry.Do(ctx, r.factory(), func(ctx context.Context) error {
		err := r.engine.Build(ctx, data)
		if err != nil {
			errs = append(errs, err)
			return retry.RetryableError(err)
		}

		return nil
	})
	if err == nil {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "built successfully")
		return nil
	}

	for _, err := range errs {
		var se workererrors.StatusError
		if !errors.As(err, &se) || se.Status == types.SubmissionStatusErrored {
			err = errors.Join(errs...)
			span.RecordError(err)
			span.SetStatus(codes.Error, "encounter non status error bubbling")
			return ErrBuildingErrored
		}
	}

	err = errors.Join(errs...)
	span.RecordError(err)
	span.SetStatus(codes.Error, "failed to build")
	return workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, ErrBuildingFailed)
}

// Check implements Engine.
func (r *BuildRetryEngine) Check(ctx context.Context, data *Params) error {
	return r.engine.Check(ctx, data)
}

// RunPov implements Engine.
func (r *BuildRetryEngine) RunPov(
	ctx context.Context,
	data *Params,
	triggerPath string,
	shouldCrash bool,
) error {
	return r.engine.RunPov(ctx, data, triggerPath, shouldCrash)
}

// RunTests implements Engine.
func (r *BuildRetryEngine) RunTests(ctx context.Context, data *Params, shouldPass bool) error {
	return r.engine.RunTests(ctx, data, shouldPass)
}
