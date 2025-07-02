package tracing

import (
	"go.opentelemetry.io/otel/trace"
)

// Set in SetupOpenTelemetry in server/otel/setup.go
var Tracer trace.Tracer
