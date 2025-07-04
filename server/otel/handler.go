package otel

import (
	"context"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/palantir/policy-bot/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type traceHandler struct {
	handler githubapp.EventHandler
}

var _ githubapp.EventHandler = (*traceHandler)(nil)

func (h *traceHandler) Handles() []string {
	return h.handler.Handles()
}

func (h *traceHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	ctx, span := tracing.Tracer.Start(ctx, "githubapp.EventHandler.Handle",
		trace.WithAttributes(
			attribute.String("event_type", eventType),
			attribute.String("delivery_id", deliveryID),
		))
	defer span.End()

	if err := h.handler.Handle(ctx, eventType, deliveryID, payload); err != nil {
		span.RecordError(err)
	}
	return nil
}

func Trace(handler githubapp.EventHandler) githubapp.EventHandler {
	return &traceHandler{handler: handler}
}
