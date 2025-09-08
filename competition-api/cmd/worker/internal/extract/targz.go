package extract

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command"
)

// Ensure TarGzExtractor implements Extractor interface.
var _ Extractor = (*TarGzExtractor)(nil)

// .tar.gz extractor
type TarGzExtractor struct {
	executor command.Executor
}

func NewTarGzExtractor(executor command.Executor) *TarGzExtractor {
	return &TarGzExtractor{
		executor: executor,
	}
}

func (e *TarGzExtractor) Extract(ctx context.Context, reader io.Reader, outDir string) error {
	ctx, span := tracer.Start(ctx, "TarExtractor.Extract", trace.WithAttributes(
		attribute.String("outDir", outDir),
	))
	defer span.End()

	cmd := command.New(
		"tar",
		"-x",
		"-z",
		"-f",
		"-",
		"-C",
		outDir,
	)
	cmd.Stdin = reader

	result, err := e.executor.Execute(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to execute tar command")
		return err
	}
	if result.ExitCode != 0 {
		err = fmt.Errorf(
			"failed to extract: exit code(%d)\nstdout(%s)\nstderr(%s)",
			result.ExitCode,
			result.Stdout,
			result.Stderr,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "nonzero exit code extracting")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "extracted tar")
	return nil
}
