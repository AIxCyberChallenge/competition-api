package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/cmds"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	otelcompetitionapi "github.com/aixcyberchallenge/competition-api/competition-api/internal/otel"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var tracer = otel.Tracer("github.com/aixcyberchallenge/competition-api/competition-api/worker")

func runApp(ctx context.Context) int {
	useOTLP, err := strconv.ParseBool(os.Getenv("USE_OTLP"))
	if err != nil {
		logger.Logger.Warn("USE_OTLP env var is invalid", "error", err)
		useOTLP = false
	}

	shutdown, err := otelcompetitionapi.SetupOTelSDK(ctx, useOTLP)
	if err != nil {
		logger.Logger.Warn("failed to setup otel sdk")
	}
	defer func() {
		fail := shutdown(ctx)
		if fail != nil {
			logger.Logger.Warn("no clean shutdown for otel", "error", fail)
		}
	}()

	carrier := otelcompetitionapi.CreateEnvCarrier()
	extractedContext := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	ctx, span := tracer.Start(
		ctx,
		"Worker",
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(extractedContext)),
	)
	defer span.End()

	err = cmds.Execute(ctx)
	if err != nil {
		logger.Logger.Error("error executing subcommands", "error", err)

		var ee workererrors.ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		return types.ExitErrored
	}

	return 0
}

func main() {
	logger.LogLevel.Set(slog.LevelDebug)
	logger.InitSlog()

	ctx := context.Background()

	os.Exit(runApp(ctx))
}
