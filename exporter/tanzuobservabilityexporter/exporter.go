// Copyright The OpenTelemetry Authors
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

package tanzuobservabilityexporter

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

const (
	defaultApplicationName = "defaultApp"
	defaultServiceName     = "defaultService"
	labelApplication       = "application"
	labelError             = "error"
	labelEventName         = "name"
	labelService           = "service"
	labelSpanKind          = "span.kind"
	labelStatusMessage     = "status.message"
	labelStatusCode        = "status.code"
)

// spanSender Interface for sending tracing spans to Tanzu Observability
type spanSender interface {
	// SendSpan sends a tracing span to Tanzu Observability.
	// traceId, spanId, parentIds and preceding spanIds are expected to be UUID strings.
	// parents and preceding spans can be empty for a root span.
	// span tag keys can be repeated (example: "user"="foo" and "user"="bar")
	// span logs are currently omitted
	SendSpan(name string, startMillis, durationMillis int64, source, traceID, spanID string, parents, followsFrom []string, tags []senders.SpanTag, spanLogs []senders.SpanLog) error
	Flush() error
	Close()
}

type exporter struct {
	cfg    *Config
	sender spanSender
	logger *zap.Logger
}

func newExporter(l *zap.Logger, c config.Exporter) (*exporter, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config: %#v", c)
	}

	l.Info("Creating Tanzu Observability exporter", zap.String("tracing_endpoint", cfg.Traces.Endpoint))
	endpoint, err := url.Parse(cfg.Traces.Endpoint)
	if nil != err {
		return nil, err
	}
	tracingPort, err := strconv.Atoi(endpoint.Port())
	if nil != err {
		return nil, err
	}

	s, err := senders.NewProxySender(&senders.ProxyConfiguration{
		Host:                 endpoint.Hostname(),
		MetricsPort:          2878,
		TracingPort:          tracingPort,
		FlushIntervalSeconds: 1,
	})
	if nil != err {
		return nil, err
	}

	return &exporter{
		cfg:    cfg,
		sender: s,
		logger: l,
	}, nil
}

func (e exporter) pushTraceData(ctx context.Context, td pdata.Traces) error {
	var errs []error

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rspans := td.ResourceSpans().At(i)
		resource := rspans.Resource()
		for j := 0; j < rspans.InstrumentationLibrarySpans().Len(); j++ {
			ispans := rspans.InstrumentationLibrarySpans().At(j)
			transform := newTraceTransformer(resource, e.cfg)
			for k := 0; k < ispans.Spans().Len(); k++ {
				select {
				case <-ctx.Done():
					return consumererror.Combine(append(errs, errors.New("context canceled")))
				default:
					transformedSpan, err := transform.Span(ispans.Spans().At(k))
					if err != nil {
						errs = append(errs, err)
						continue
					}

					if err := e.RecordSpan(transformedSpan); err != nil {
						errs = append(errs, err)
						continue
					}
				}
			}
		}
	}

	return consumererror.Combine(errs)
}

func (e exporter) Shutdown(_ context.Context) error {
	e.sender.Close()
	return nil
}

func (e exporter) RecordSpan(span Span) error {
	var parents []string
	if span.ParentSpanID != uuid.Nil {
		parents = []string{span.ParentSpanID.String()}
	}

	err := e.sender.SendSpan(
		span.Name,
		span.StartMillis,
		span.DurationMillis,
		"",
		span.TraceID.String(),
		span.SpanID.String(),
		parents,
		nil,
		mapToSpanTags(span.Tags),
		span.SpanLogs,
	)

	if err != nil {
		return err
	}
	return e.sender.Flush()
}

func mapToSpanTags(tags map[string]string) []senders.SpanTag {
	var spanTags []senders.SpanTag
	for k, v := range tags {
		spanTags = append(spanTags, senders.SpanTag{
			Key:   k,
			Value: v,
		})
	}
	return spanTags
}
