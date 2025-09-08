package fetch

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch",
)

//go:generate mockgen -destination ./mock/mock.go -package mock . Fetcher

type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}
