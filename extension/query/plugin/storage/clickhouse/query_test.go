package clickhouse

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestBuildQuery(t *testing.T) {

}

func TestSearchTraces(t *testing.T) {
	factory := NewFactory(&ct)
	require.NotNil(t, factory)

	err := factory.Initialize(&zap.Logger{})
	require.NoError(t, err)

	//err = createTracesTable(factory.client)
	//require.NoError(t, err)
	//
	//insertTracesDate(factory.client)

	//defer func() {
	//	deleteTracesTables(factory.client)
	//}()

	query, err := factory.CreateSpanQuery()
	require.NotNil(t, query)
	require.NoError(t, err)

	var req = storage.TraceQueryParameters{
		ServiceName:   "",
		OperationName: "",
		Tags:          nil,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		DurationMin:   nil,
		DurationMax:   nil,
		NumTraces:     0,
	}
	resp, err := query.SearchTraces(context.Background(), &req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

const (
	// language=ClickHouse SQL
	createTracesTableSQL = `
CREATE TABLE IF NOT EXISTS %s (
     Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
     TraceId String CODEC(ZSTD(1)),
     SpanId String CODEC(ZSTD(1)),
     ParentSpanId String CODEC(ZSTD(1)),
     TraceState String CODEC(ZSTD(1)),
     SpanName LowCardinality(String) CODEC(ZSTD(1)),
     SpanKind LowCardinality(String) CODEC(ZSTD(1)),
     ServiceName LowCardinality(String) CODEC(ZSTD(1)),
     ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     SpanAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     Duration Int64 CODEC(ZSTD(1)),
     StatusCode LowCardinality(String) CODEC(ZSTD(1)),
     StatusMessage String CODEC(ZSTD(1)),
     Events Nested (
         Timestamp DateTime64(9),
         Name LowCardinality(String),
         Attributes Map(LowCardinality(String), String)
     ) CODEC(ZSTD(1)),
     Links Nested (
         TraceId String,
         SpanId String,
         TraceState String,
         Attributes Map(LowCardinality(String), String)
     ) CODEC(ZSTD(1)),
     INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
     INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_span_attr_key mapKeys(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_span_attr_value mapValues(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_duration Duration TYPE minmax GRANULARITY 1
) ENGINE MergeTree()
%s
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toUnixTimestamp(Timestamp), TraceId)
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	insertTracesSQLTemplate = `INSERT INTO %s (
                        Timestamp,
                        TraceId,
                        SpanId,
                        ParentSpanId,
                        TraceState,
                        SpanName,
                        SpanKind,
                        ServiceName,
                        ResourceAttributes,
                        SpanAttributes,
                        Duration,
                        StatusCode,
                        StatusMessage,
                        Events.Timestamp,
                        Events.Name,
                        Events.Attributes,
                        Links.TraceId,
                        Links.SpanId,
                        Links.TraceState,
                        Links.Attributes
                        )`
)

func createTracesTable(conn clickhouse.Conn) error {
	ctx := context.Background()
	if err := conn.Exec(ctx, `truncate table IF EXISTS otel.otel_traces`); err != nil {
		return err
	}
	//err := conn.Exec(ctx, fmt.Sprintf(createTracesTableSQL, "otel.otel_traces", "3"))
	//if err != nil {
	//	return err
	//}
	return nil
}

func insertTracesDate(conn clickhouse.Conn) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, fmt.Sprintf(insertTracesSQLTemplate, "otel.otel_traces"))
	if err != nil {
		return err
	}

	td := simpleTraces(10)
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		spans := td.ResourceSpans().At(i)
		res := spans.Resource()
		resAttr := attributesToMap(res.Attributes())
		var serviceName string
		if v, ok := res.Attributes().Get(conventions.AttributeServiceName); ok {
			serviceName = v.Str()
		}

		for j := 0; j < spans.ScopeSpans().Len(); j++ {
			rs := spans.ScopeSpans().At(j).Spans()
			for k := 0; k < rs.Len(); k++ {
				r := rs.At(k)
				spanAttr := attributesToMap(r.Attributes())
				// ?
				serviceName = spanAttr["service.name"]
				status := r.Status()
				eventTimes, eventNames, eventAttrs := convertEvents_(r.Events())
				linksTraceIDs, linksSpanIDs, linksTraceStates, linksAttrs := convertLinks_(r.Links())
				err = batch.Append(r.StartTimestamp().AsTime(),
					r.TraceID().HexString(),
					r.SpanID().HexString(),
					r.ParentSpanID().HexString(),
					r.TraceState().AsRaw(),
					r.Name(),
					r.Kind().String(),
					serviceName,
					resAttr,
					spanAttr,
					r.EndTimestamp().AsTime().Sub(r.StartTimestamp().AsTime()).Nanoseconds(),
					status.Code().String(),
					status.Message(),
					eventTimes,
					eventNames,
					eventAttrs,
					linksTraceIDs,
					linksSpanIDs,
					linksTraceStates,
					linksAttrs,
				)
				if err != nil {
					return err
				}
			}
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	return nil
}

func simpleTraces(count int) ptrace.Traces {
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	for i := 0; i < count; i++ {
		s := ss.Spans().AppendEmpty()
		s.SetTraceID(uInt64ToTraceID(0, 1))
		s.SetSpanID(uInt64ToSpanID(1))
		s.TraceState().FromRaw("TraceState")
		s.SetParentSpanID(uInt64ToSpanID(1))
		s.SetName("span_name xxx")
		s.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		s.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		s.Attributes().PutStr("service.name", "this service name")
		s.Attributes().PutStr("a1", "v1")
		s.Attributes().PutStr("a2", "v2")
		s.Status().SetCode(ptrace.StatusCodeOk)
		s.Status().SetMessage("sucess Message")

		s.SetDroppedAttributesCount(3)
		s.SetDroppedEventsCount(2)
		s.SetDroppedLinksCount(1)

		event := s.Events().AppendEmpty()
		event.SetName("event1")
		event.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		event.Attributes().PutStr("event_attrubute_1", "value1")
		event.Attributes().PutStr("event_attrubute_2", "value2")

		event1 := s.Events().AppendEmpty()
		event1.SetName("event2")
		event1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		event1.Attributes().PutStr("event2_attrubute_1", "value1")
		event1.Attributes().PutStr("event2_attrubute_2", "value2")

		link := s.Links().AppendEmpty()
		link.SetTraceID(uInt64ToTraceID(0, 1))
		link.SetSpanID(uInt64ToSpanID(1))
		link.TraceState().FromRaw("TraceState")
		link.Attributes().PutStr("k", "v")

		link2 := s.Links().AppendEmpty()
		link2.SetTraceID(uInt64ToTraceID(0, 1))
		link2.SetSpanID(uInt64ToSpanID(1))
		link2.TraceState().FromRaw("TraceState2")
		link2.Attributes().PutStr("k2", "v2")
	}
	return traces
}

func deleteTracesTables(conn clickhouse.Conn) {
	conn.Exec(context.Background(), "DROP TABLE otel.otel_traces")
}

func attributesToMap(attributes pcommon.Map) map[string]string {
	m := make(map[string]string, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		m[k] = v.Str()
		return true
	})
	return m
}

func convertEvents_(events ptrace.SpanEventSlice) ([]time.Time, []string, []map[string]string) {
	var (
		times []time.Time
		names []string
		attrs []map[string]string
	)
	for i := 0; i < events.Len(); i++ {
		event := events.At(i)
		times = append(times, event.Timestamp().AsTime())
		names = append(names, event.Name())
		attrs = append(attrs, attributesToMap(event.Attributes()))
	}
	return times, names, attrs
}

func convertLinks_(links ptrace.SpanLinkSlice) ([]string, []string, []string, []map[string]string) {
	var (
		traceIDs []string
		spanIDs  []string
		states   []string
		attrs    []map[string]string
	)
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		traceIDs = append(traceIDs, link.TraceID().HexString())
		spanIDs = append(spanIDs, link.SpanID().HexString())
		states = append(states, link.TraceState().AsRaw())
		attrs = append(attrs, attributesToMap(link.Attributes()))
	}
	return traceIDs, spanIDs, states, attrs
}

func uInt64ToTraceID(high, low uint64) pcommon.TraceID {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return traceID
}

func uInt64ToSpanID(id uint64) pcommon.SpanID {
	spanID := [8]byte{}
	binary.BigEndian.PutUint64(spanID[:], id)
	return pcommon.SpanID(spanID)
}
