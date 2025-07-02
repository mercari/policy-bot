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
			semconv.ServiceName("policy-bot"),
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
	shutdownFuncs = append(shutdownFuncs, texporter.Shutdown)

	tprovider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(texporter),
	)
	shutdownFuncs = append(shutdownFuncs, tprovider.Shutdown)

	otel.SetTracerProvider(tprovider)

	return
}
