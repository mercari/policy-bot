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
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type ZerologOtelGoogleCloudHook struct{}

func (*ZerologOtelGoogleCloudHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	sc := trace.SpanContextFromContext(e.GetCtx())
	if !sc.IsValid() {
		return
	}

	e.Str("logging.googleapis.com/spanId", sc.SpanID().String()).
		Str("logging.googleapis.com/trace", sc.TraceID().String()).
		Bool("logging.googleapis.com/trace_sampled", sc.IsSampled())
}
