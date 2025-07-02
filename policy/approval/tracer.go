package approval

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("github.com/mercari/policy-bot/policy/approval")
