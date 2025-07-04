// Copyright 2025 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otel

import (
	"context"
	"errors"
	"fmt"

	"github.com/palantir/policy-bot/tracing"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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
			semconv.ServiceName("policy-bot"),
		),
	}

	res, err := resource.New(ctx, resOpts...)
	if err != nil {
		handleErr(fmt.Errorf("failed to create OpenTelemetry resource: %w", err))
		return
	}

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	var texporter sdktrace.SpanExporter
	if googleCloudSupport {
		var creds credentials.PerRPCCredentials
		creds, err = oauth.NewApplicationDefault(ctx)
		if err != nil {
			handleErr(fmt.Errorf("failed to create Google Cloud credentials: %w", err))
			return
		}

		texporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithDialOption(grpc.WithPerRPCCredentials(creds)))
		if err != nil {
			handleErr(fmt.Errorf("failed to create OpenTelemetry exporter: %w", err))
			return
		}
	} else {
		texporter, err = autoexport.NewSpanExporter(context.Background())
		if err != nil {
			handleErr(fmt.Errorf("failed to create OpenTelemetry exporter: %w", err))
			return
		}
	}
	shutdownFuncs = append(shutdownFuncs, texporter.Shutdown)

	tprovider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(texporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	shutdownFuncs = append(shutdownFuncs, tprovider.Shutdown)
	otel.SetTracerProvider(tprovider)

	tracing.Tracer = tprovider.Tracer("github.com/palantir/policy-bot")

	return
}
