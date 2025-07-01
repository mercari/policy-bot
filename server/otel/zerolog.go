package otel

import (
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type OtelZerologGoogleCloudHook struct{}

func (*OtelZerologGoogleCloudHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	sc := trace.SpanContextFromContext(e.GetCtx())
	if !sc.IsValid() {
		return
	}

	e.Str("logging.googleapis.com/spanId", sc.SpanID().String()).
		Str("logging.googleapis.com/trace", sc.TraceID().String()).
		Bool("logging.googleapis.com/trace_sampled", sc.IsSampled())
}
