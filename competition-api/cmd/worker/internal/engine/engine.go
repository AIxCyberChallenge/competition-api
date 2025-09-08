package engine

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/aixcyberchallenge/competition-api/worker/internal/engine")

//go:generate mockgen -destination ./mock/mock.go -package mock . Engine

type Engine interface {
	Check(ctx context.Context, data *Params) error
	Build(ctx context.Context, data *Params) error
	RunPov(
		ctx context.Context,
		data *Params,
		triggerPath string,
		shouldCrash bool,
	) error
	ApplyPatch(
		ctx context.Context,
		data *Params,
		patchPath string,
	) error
	RunTests(
		ctx context.Context,
		data *Params,
		shouldPass bool,
	) error
}

var (
	ErrBuildingFailed   = errors.New("building failed")
	ErrBuildingErrored  = errors.New("building errored")
	ErrAptUnreachable   = errors.New("failed to reach apt")
	ErrMavenUnreachable = errors.New("failed to reach maven")
)
