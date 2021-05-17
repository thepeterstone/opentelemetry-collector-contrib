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
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wavefronthq/wavefront-sdk-go/senders"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

type traceTransformer struct {
	ResourceAttributes pdata.AttributeMap
	Config             *Config
}

func newTraceTransformer(resource pdata.Resource, cfg *Config) *traceTransformer {
	t := &traceTransformer{
		ResourceAttributes: resource.Attributes(),
		Config:             cfg,
	}
	return t
}

var (
	errInvalidSpanID  = errors.New("SpanID is invalid")
	errInvalidTraceID = errors.New("TraceID is invalid")
)

type Span struct {
	Name           string
	TraceID        uuid.UUID
	SpanID         uuid.UUID
	ParentSpanID   uuid.UUID
	Tags           map[string]string
	StartMillis    int64
	DurationMillis int64
	SpanLogs       []senders.SpanLog
}

func (t *traceTransformer) Span(orig pdata.Span) (Span, error) {
	traceID, err := traceIDtoUUID(orig.TraceID())
	if err != nil {
		return Span{}, errInvalidTraceID
	}

	spanID, err := spanIDtoUUID(orig.SpanID())
	if err != nil {
		return Span{}, errInvalidSpanID
	}

	parentSpanID, err := parentSpanIDtoUUID(orig.ParentSpanID())
	if err != nil {
		return Span{}, errInvalidSpanID
	}

	startMillis, durationMillis := calculateTimes(orig)

	tags := attributesToTags(t.ResourceAttributes, orig.Attributes())
	t.setRequiredTags(tags)

	tags[labelSpanKind] = spanKind(orig)

	errorTags := errorTagsFromStatus(orig.Status())
	for k, v := range errorTags {
		tags[k] = v
	}

	if len(orig.TraceState()) > 0 {
		tags[tracetranslator.TagW3CTraceState] = string(orig.TraceState())
	}

	return Span{
		Name:           orig.Name(),
		TraceID:        traceID,
		SpanID:         spanID,
		ParentSpanID:   parentSpanID,
		Tags:           tags,
		StartMillis:    startMillis,
		DurationMillis: durationMillis,
		SpanLogs:       eventsToLogs(orig.Events()),
	}, nil
}

func spanKind(span pdata.Span) string {
	switch span.Kind() {
	case pdata.SpanKindCLIENT:
		return "client"
	case pdata.SpanKindSERVER:
		return "server"
	case pdata.SpanKindPRODUCER:
		return "producer"
	case pdata.SpanKindCONSUMER:
		return "consumer"
	case pdata.SpanKindINTERNAL:
		return "internal"
	case pdata.SpanKindUNSPECIFIED:
		return "unspecified"
	default:
		return "unknown"
	}
}

func (t *traceTransformer) setRequiredTags(tags map[string]string) {
	if _, ok := tags[labelService]; !ok {
		if _, svcNameOk := tags[conventions.AttributeServiceName]; svcNameOk {
			tags[labelService] = tags[conventions.AttributeServiceName]
			delete(tags, conventions.AttributeServiceName)
		} else {
			tags[labelService] = defaultServiceName
		}
	}
	if _, ok := tags[labelApplication]; !ok {
		tags[labelApplication] = defaultApplicationName
	}
}

func eventsToLogs(events pdata.SpanEventSlice) []senders.SpanLog {
	var result []senders.SpanLog
	for i := 0; i < events.Len(); i++ {
		e := events.At(i)
		fields := attributesToTags(e.Attributes())
		fields[labelEventName] = e.Name()
		result = append(result, senders.SpanLog{
			Timestamp: int64(e.Timestamp()) / time.Microsecond.Nanoseconds(), // Timestamp is in microseconds
			Fields:    fields,
		})
	}

	return result
}

func calculateTimes(span pdata.Span) (int64, int64) {
	startMillis := int64(span.StartTimestamp()) / time.Millisecond.Nanoseconds()
	endMillis := int64(span.EndTimestamp()) / time.Millisecond.Nanoseconds()
	durationMillis := endMillis - startMillis
	// it's possible end time is unset, so default to 0 rather than using a negative number
	if span.EndTimestamp() == 0 {
		durationMillis = 0
	}
	return startMillis, durationMillis
}

func attributesToTags(attributes ...pdata.AttributeMap) map[string]string {
	tags := map[string]string{}

	extractTag := func(k string, v pdata.AttributeValue) bool {
		tags[k] = tracetranslator.AttributeValueToString(v, false)
		return true
	}

	// Since AttributeMaps are processed in the order received, later values overwrite earlier ones
	for _, att := range attributes {
		att.Range(extractTag)
	}

	return tags
}

func errorTagsFromStatus(status pdata.SpanStatus) map[string]string {
	tags := map[string]string{
		labelStatusCode: fmt.Sprintf("%d", status.Code()),
	}
	if status.Code() != pdata.StatusCodeError {
		return tags
	}

	tags[labelError] = "true"
	if status.Message() != "" {
		msg := status.Message()
		maxLength := 255 - len(labelStatusMessage+"=")
		if len(msg) > maxLength {
			msg = msg[:maxLength]
		}
		tags[labelStatusMessage] = msg
	}
	return tags
}

func traceIDtoUUID(id pdata.TraceID) (uuid.UUID, error) {
	formatted, err := uuid.Parse(id.HexString())
	if err != nil || id.IsEmpty() {
		return uuid.Nil, errInvalidTraceID
	}
	return formatted, nil
}

func spanIDtoUUID(id pdata.SpanID) (uuid.UUID, error) {
	formatted, err := uuid.FromBytes(padTo16Bytes(id.Bytes()))
	if err != nil || id.IsEmpty() {
		return uuid.Nil, errInvalidSpanID
	}
	return formatted, nil
}

func parentSpanIDtoUUID(id pdata.SpanID) (uuid.UUID, error) {
	if id.IsEmpty() {
		return uuid.Nil, nil
	}
	formatted, err := uuid.FromBytes(padTo16Bytes(id.Bytes()))
	if err != nil {
		return uuid.Nil, errInvalidSpanID
	}
	return formatted, nil
}

func padTo16Bytes(b [8]byte) []byte {
	as16bytes := make([]byte, 16)
	copy(as16bytes[16-len(b):], b[:])
	return as16bytes
}
