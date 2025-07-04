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
