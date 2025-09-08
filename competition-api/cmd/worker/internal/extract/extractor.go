package extract

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/worker/internal/extract",
)

//go:generate mockgen -destination ./mock/mock.go -package mock . Extractor

// Extract archive to a directory
type Extractor interface {
	Extract(ctx context.Context, reader io.Reader, outDir string) error
}
