package datasource

import (
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	v1_common "go.opentelemetry.io/proto/otlp/common/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"testing"
	"time"
)

var (
	TestSpanStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestSpanStartTimestamp = pcommon.NewTimestampFromTime(TestSpanStartTime)

	TestSpanEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestSpanEventTimestamp = pcommon.NewTimestampFromTime(TestSpanEventTime)

	TestSpanEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestSpanEndTimestamp = pcommon.NewTimestampFromTime(TestSpanEndTime)
)

var (
	TraceId      = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	ParentSpanId = []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8}
)

func TestFindRootSpan(t *testing.T) {
	td := GenerateTracesWithTwoSpan()
	rootSpan := FindRootSpan(td.ResourceSpans[0].ScopeSpans[0].Spans)
	assert.Equal(t, string(rootSpan.SpanId), string([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8}))
}

func fillSpanOne() *v1_trace.Span {
	span := &v1_trace.Span{}
	span.Name = "operationA"
	span.StartTimeUnixNano = uint64(TestSpanStartTimestamp)
	span.EndTimeUnixNano = uint64(TestSpanEndTimestamp)
	span.DroppedAttributesCount = 1
	span.Events = []*v1_trace.Span_Event{
		{
			TimeUnixNano: uint64(TestSpanEventTimestamp),
			Name:         "event-with-attr",
			Attributes: []*v1_common.KeyValue{
				{
					Key:   "span-event-attr",
					Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: "span-event-attr-val"}},
				},
			},
			DroppedAttributesCount: 2,
		},
		{
			TimeUnixNano:           uint64(TestSpanEventTimestamp),
			Name:                   "event",
			DroppedAttributesCount: 2,
		},
	}

	span.DroppedEventsCount = 1
	span.Status = &v1_trace.Status{
		Code:    v1_trace.Status_STATUS_CODE_ERROR,
		Message: "status-cancelled",
	}
	return span
}

func GenerateTracesWithTwoSpan() *v1_trace.TracesData {

	parentSpan := fillSpanOne()
	parentSpan.Name = "parent-span-name"
	parentSpan.TraceId = TraceId
	parentSpan.SpanId = ParentSpanId

	childSpan := fillSpanOne()
	childSpan.Name = "child-span-name"
	childSpan.TraceId = TraceId
	childSpan.SpanId = []byte{0xFF, 0xFF, 0xFD, 0xFC, 0xFB, 0xFA, 0xFD, 0xF8}
	childSpan.ParentSpanId = ParentSpanId

	td := &v1_trace.TracesData{ResourceSpans: []*v1_trace.ResourceSpans{
		{
			ScopeSpans: []*v1_trace.ScopeSpans{
				{
					Spans: []*v1_trace.Span{
						childSpan, parentSpan,
					},
				},
			},
		},
	}}
	return td
}
