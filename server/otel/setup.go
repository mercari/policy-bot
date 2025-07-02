package otel

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func SetupOpenTelemetry(ctx context.Context, logger zerolog.Logger, googleCloudSupport bool) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		logger.Info().Msg("Shutting down OpenTelemetry")
		errs := make([]error, 0)
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	resOpts := []resource.Option{
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("policy-bot"),
		),
	}
	if googleCloudSupport {
		resOpts = append(resOpts, resource.WithDetectors(gcp.NewDetector()))
	}

	res, err := resource.New(ctx, resOpts...)
	if err != nil {
		handleErr(fmt.Errorf("failed to create OpenTelemetry resource: %w", err))
		return
	}

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())
	texporter, err := autoexport.NewSpanExporter(context.Background())
	if err != nil {
		handleErr(fmt.Errorf("failed to create OpenTelemetry exporter: %w", err))
		return
	}
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(texporter),
	))
	return
}
