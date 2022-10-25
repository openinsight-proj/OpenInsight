package clickhouse

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
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

func TestBuildQuery(t *testing.T) {
	factory := NewFactory(&ct)
	require.NotNil(t, factory)

	err := factory.Initialize(&zap.Logger{})
	require.NoError(t, err)

	tests := []struct {
		param storage.TraceQueryParameters
	}{
		{
			param: storage.TraceQueryParameters{
				ServiceName: "this service [9]",
			},
		},
		{
			param: storage.TraceQueryParameters{
				ServiceName:   "this service [9]",
				OperationName: "HTTP PUT",
			},
		},
		{
			param: storage.TraceQueryParameters{
				ServiceName:   "this service [9]",
				OperationName: "HTTP PUT",
				Tags: map[string]string{
					"Tag_a": "tag_a_value",
					"Tag_b": "tag_b_value",
				},
			},
		},
		{
			param: storage.TraceQueryParameters{
				ServiceName:   "this service [9]",
				OperationName: "HTTP PUT",
				Tags: map[string]string{
					"Tag_a": "tag_a_value",
					"Tag_b": "tag_b_value",
				},
				StartTime: time.Now(),
				EndTime:   time.Now().Add(time.Second * 3),
			},
		},
		{
			param: storage.TraceQueryParameters{
				ServiceName:   "this service [9]",
				OperationName: "HTTP PUT",
				Tags: map[string]string{
					"Tag_a": "tag_a_value",
					"Tag_b": "tag_b_value",
				},
				StartTime:   time.Now(),
				EndTime:     time.Now().Add(time.Second * 3),
				DurationMin: durationpb.New(time.Millisecond * 20),
				DurationMax: durationpb.New(time.Millisecond * 100),
			},
		},
		{
			param: storage.TraceQueryParameters{
				ServiceName:   "this service [9]",
				OperationName: "HTTP PUT",
				Tags: map[string]string{
					"Tag_a": "tag_a_value",
					"Tag_b": "tag_b_value",
				},
				StartTime:   time.Now(),
				EndTime:     time.Now().Add(time.Second * 3),
				DurationMin: durationpb.New(time.Millisecond * 20),
				DurationMax: durationpb.New(time.Millisecond * 100),
				NumTraces:   20,
			},
		},
	}
	for _, c := range tests {
		sql, err := buildQuery(&c.param, "otel_traces")
		require.NoError(t, err)
		require.NotNil(t, sql)
	}
}

func TestSearchTraces(t *testing.T) {

	query, err := initQuery()
	require.NoError(t, err)

	req := storage.TraceQueryParameters{
		ServiceName: "my-otel-demo-frontend",
		//2022-10-23 16:43:08
		StartTime: time.Date(2022, 10, 25, 13, 58, 47, 0, time.UTC),
		//2022-10-23 16:43:14
		EndTime: time.Date(2022, 10, 25, 14, 58, 14, 0, time.UTC),
	}
	resp, err := query.SearchTraces(context.Background(), &req)
	require.NoError(t, err)
	require.NotNil(t, resp)

}

func TestGetService(t *testing.T) {
	query, err := initQuery()
	require.NoError(t, err)

	resp, err := query.GetService(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestGetTrace(t *testing.T) {
	query, err := initQuery()
	require.NoError(t, err)

	resp, err := query.GetTrace(context.Background(), "00000000000000000000000000000001")
	require.NoError(t, err)
	require.NotNil(t, resp)

}

func initQuery() (storage.Query, error) {
	factory := NewFactory(&ct)

	err := factory.Initialize(&zap.Logger{})
	if err != nil {
		return nil, err
	}

	//err = truncateTracesTable(factory.client)
	//if err != nil {
	//	return nil, err
	//}
	//
	//err = insertTracesDate(factory.client)
	//if err != nil {
	//	return nil, err
	//}

	query, err := factory.CreateSpanQuery()
	if err != nil {
		return nil, err
	}
	return query, err
}

func truncateTracesTable(conn clickhouse.Conn) error {
	ctx := context.Background()
	if err := conn.Exec(ctx, `truncate table IF EXISTS otel.otel_traces`); err != nil {
		return err
	}
	if err := conn.Exec(ctx, `truncate table IF EXISTS otel.otel_traces_trace_id_ts`); err != nil {
		return err
	}
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
	rs.Resource().Attributes().PutStr("Resource_Attributes_key_1", "value1")
	rs.Resource().Attributes().PutStr("Resource_Attributes_key_2", "value2")
	ss := rs.ScopeSpans().AppendEmpty()
	for i := 0; i < count; i++ {
		s := ss.Spans().AppendEmpty()
		s.SetTraceID(uInt64ToTraceID(0, uint64(i)))
		s.SetSpanID(uInt64ToSpanID(uint64(i)))
		s.TraceState().FromRaw("TraceState")
		s.SetParentSpanID(uInt64ToSpanID(uint64(i)))
		s.SetName("span_name xxx")
		s.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		s.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(time.Minute * 5)))
		s.Attributes().PutStr("service.name", fmt.Sprintf("this service [%d]", i))
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
		link.SetTraceID(uInt64ToTraceID(0, uint64(i)))
		link.SetSpanID(uInt64ToSpanID(uint64(i)))
		link.TraceState().FromRaw("TraceState")
		link.Attributes().PutStr("k", "v")

		link2 := s.Links().AppendEmpty()
		link2.SetTraceID(uInt64ToTraceID(0, uint64(i)))
		link2.SetSpanID(uInt64ToSpanID(uint64(i)))
		link2.TraceState().FromRaw("TraceState2")
		link2.Attributes().PutStr("k2", "v2")

		// span 2
		s2 := ss.Spans().AppendEmpty()
		s2.SetTraceID(uInt64ToTraceID(0, uint64(i)))
		s2.SetSpanID(uInt64ToSpanID(uint64(3)))
		s2.TraceState().FromRaw("TraceState")
		s2.SetParentSpanID(uInt64ToSpanID(uint64(i)))
		s2.SetName("span_name xxx/ccc")
		s2.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		s2.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(time.Minute * 5)))
		s2.Attributes().PutStr("service.name", fmt.Sprintf("this service [%d]", i))
		s2.Attributes().PutStr("a1", "v1")
		s2.Attributes().PutStr("a2", "v2")
		s2.Status().SetCode(ptrace.StatusCodeOk)
		s2.Status().SetMessage("sucess Message")

		s2.SetDroppedAttributesCount(3)
		s2.SetDroppedEventsCount(2)
		s2.SetDroppedLinksCount(1)
		time.Sleep(time.Second)
	}
	return traces
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
	return spanID
}
