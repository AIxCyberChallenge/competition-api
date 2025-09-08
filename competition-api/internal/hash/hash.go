package hash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/aixcyberchallenge/competition-api/competition-api/internal/hash")

// Will consume reader to the end
func Reader(ctx context.Context, f io.Reader) (string, error) {
	_, span := tracer.Start(ctx, "Reader")
	defer span.End()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to copy file into hasher")
		return "", err
	}

	sum := hex.EncodeToString(h.Sum(nil))

	span.AddEvent("digested", trace.WithAttributes(attribute.String("sum", sum)))

	return sum, nil
}

func Buffer(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
