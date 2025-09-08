package otel

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(
	ctx context.Context,
	useOTLP bool,
) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var er error
		for _, fn := range shutdownFuncs {
			er = errors.Join(er, fn(ctx))
		}
		shutdownFuncs = nil
		return er
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) error {
		return errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTracerProvider(useOTLP)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(useOTLP)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	loggerProvider, err := newLoggerProvider(useOTLP)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return shutdown, nil
}

//nolint:ireturn // no control over otel's propagator interface return.
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(useOTLP bool) (*trace.TracerProvider, error) {
	var err error
	var traceExporter trace.SpanExporter

	if useOTLP {
		traceExporter, err = otlptracegrpc.New(context.Background())
	} else {
		traceExporter, err = stdouttrace.New()
	}
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(traceExporter),
	)
	return tracerProvider, nil
}

func newMeterProvider(useOTLP bool) (*metric.MeterProvider, error) {
	var err error
	var metricExporter metric.Exporter

	if useOTLP {
		metricExporter, err = otlpmetricgrpc.New(context.Background())
	} else {
		metricExporter, err = stdoutmetric.New()
	}
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
	)
	return meterProvider, nil
}

func newLoggerProvider(useOTLP bool) (*log.LoggerProvider, error) {
	var err error
	var logExporter log.Exporter

	if useOTLP {
		logExporter, err = otlploggrpc.New(context.Background())
	} else {
		logExporter, err = stdoutlog.New()
	}
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
