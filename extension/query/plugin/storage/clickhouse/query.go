package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"strings"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_common "go.opentelemetry.io/proto/otlp/common/v1"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
)

type ClickHouseQuery struct {
	logger           *zap.Logger
	client           clickhouse.Conn
	loggingTableName string
	tracingTableName string
	metricsTableName string
}

type TracesModel struct {
	Timestamp          time.Time           `ch:"Timestamp"`
	TraceId            string              `ch:"TraceId"`
	SpanId             string              `ch:"SpanId"`
	ParentSpanId       string              `ch:"ParentSpanId"`
	TraceState         string              `ch:"TraceState"`
	SpanName           string              `ch:"SpanName"`
	SpanKind           string              `ch:"SpanKind"`
	ServiceName        string              `ch:"ServiceName"`
	ResourceAttributes map[string]string   `ch:"ResourceAttributes"`
	SpanAttributes     map[string]string   `ch:"SpanAttributes"`
	Duration           int64               `ch:"Duration"`
	StatusCode         string              `ch:"StatusCode"`
	StatusMessage      string              `ch:"StatusMessage"`
	EventsTimestamp    []time.Time         `ch:"Events.Timestamp"`
	EventsName         []string            `ch:"Events.Name"`
	EventsAttributes   []map[string]string `ch:"Events.Attributes"`
	LinksTraceId       []string            `ch:"Links.TraceId"`
	LinksSpanId        []string            `ch:"Links.SpanId"`
	LinksTraceState    []string            `ch:"Links.TraceState"`
	LinksAttributes    []map[string]string `ch:"Links.Attributes"`
	Start              time.Time           `ch:"Start"`
	End                time.Time           `ch:"End"`
}

func (q *ClickHouseQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	return &v1_trace.TracesData{}, nil
}

func (q *ClickHouseQuery) SearchTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {
	sql, err := buildQuery(query, q.tracingTableName)
	if err != nil {
		return nil, err
	}
	var result []TracesModel
	if err := q.client.Select(ctx, &result, sql); err != nil {
		return nil, err
	}
	return convertSpan(result), nil
}

func (q *ClickHouseQuery) SearchLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}
func (q *ClickHouseQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func buildQuery(query *storage.TraceQueryParameters, tableName string) (string, error) {

	//todo build SQL
	// otel_traces_trace_id_ts -> tableName_trace_id_ts
	base := `SELECT a.Timestamp,
       a.TraceId,
       a.SpanId,
       a.ParentSpanId,
       a.SpanName,
       a.SpanKind,
       a.ServiceName,
       a.Duration,
       a.StatusCode,
       a.StatusMessage,
       a.SpanAttributes,
       a.ResourceAttributes,
       a.Events.Timestamp,
       a.Events.Name,
       a.Events.Attributes,
       a.Links.TraceId,
       a.Links.SpanId,
       a.Links.TraceState,
       a.Links.Attributes,
       b.Start,
       b.End FROM otel_traces AS a JOIN otel_traces_trace_id_ts AS b ON a.TraceId = b.TraceId`

	fmt.Println(base)

	limit := `SELECT a.TraceId FROM otel_traces AS a JOIN otel_traces_trace_id_ts AS b ON a.TraceId = b.TraceId`

	var whereList []string
	if query.ServiceName == "" {
		return "", errors.New("query parameter must contain  ServiceName")
	}
	whereList = append(whereList, fmt.Sprintf("a.ServiceName='%s'", query.ServiceName))
	if query.OperationName != "" {
		whereList = append(whereList, fmt.Sprintf("a.SpanName='%s'", query.ServiceName))
	}
	if len(query.Tags) != 0 {
		for key, value := range query.Tags {
			whereList = append(whereList, fmt.Sprintf("a.SpanAttributes['%s']='%s'", key, value))
		}
	}
	//StartTime <= b.Start <= EndTime
	if query.EndTime.After(query.StartTime) {
		whereList = append(whereList, fmt.Sprintf("'%s'<=b.Start", query.StartTime.Format("2019-01-01 00:00:00")))
		whereList = append(whereList, fmt.Sprintf("b.Start<='%s'", query.StartTime.Format("2019-01-01 00:00:00")))
	}
	//DurationMin <= a.Duration <= DurationMax
	if query.DurationMin.Nanos < query.DurationMax.Nanos {
		whereList = append(whereList, fmt.Sprintf("%d<=a.Duration", query.DurationMin.Nanos))
		whereList = append(whereList, fmt.Sprintf("a.Duration=<%d", query.DurationMax.Nanos))
	}
	whereCondition := fmt.Sprintf("WHERE %s", strings.Join(whereList, "AND"))

	//add where
	if len(whereList) != 0 {
		limit = limit + " " + whereCondition
	}

	//todo
	// limit := fmt.Sprintf("LIMIT %d", query.NumTraces)
	// SELECT TraceId FROM otel_traces_trace_id_ts GROUP BY TraceId

	return `SELECT a.Timestamp,
       a.TraceId,
       a.SpanId,
       a.ParentSpanId,
       a.SpanName,
       a.SpanKind,
       a.ServiceName,
       a.Duration,
       a.StatusCode,
       a.StatusMessage,
       a.SpanAttributes,
       a.ResourceAttributes,
       a.Events.Timestamp,
       a.Events.Name,
       a.Events.Attributes,
       a.Links.TraceId,
       a.Links.SpanId,
       a.Links.TraceState,
       a.Links.Attributes,
       b.Start,
       b.End FROM otel_traces AS a JOIN otel_traces_trace_id_ts AS b ON a.TraceId = b.TraceId`, nil
}

func convertSpan(tracesModel []TracesModel) *v1_trace.TracesData {
	var spanSlice []*v1_trace.Span
	for _, item := range tracesModel {
		s := v1_trace.Span{}
		s.TraceId = []byte(item.TraceId)
		s.SpanId = []byte(item.SpanId)
		s.ParentSpanId = []byte(item.ParentSpanId)
		s.TraceState = item.TraceState
		s.Name = item.SpanName
		s.Kind = v1_trace.Span_SpanKind(v1_trace.Span_SpanKind_value[item.SpanKind])
		// item.ServiceName in attribute
		s.StartTimeUnixNano = uint64(item.Timestamp.UnixNano())
		duration, _ := time.ParseDuration(fmt.Sprintf("%dns", item.Duration))
		s.EndTimeUnixNano = uint64(item.Timestamp.Add(duration).UnixNano())
		s.Attributes = convertAttributes(item.SpanAttributes)
		//s.DroppedAttributesCount
		s.Events = convertEvents(item.EventsName, item.EventsTimestamp, item.EventsAttributes)
		//s.DroppedEventsCount
		s.Links = convertLinks(item.LinksTraceId, item.LinksSpanId, item.LinksTraceState, item.LinksAttributes)
		//s.DroppedLinksCount
		s.Status = &v1_trace.Status{
			Message: item.StatusMessage,
			Code:    v1_trace.Status_StatusCode(v1_trace.Span_SpanKind_value[item.StatusCode]),
		}
		spanSlice = append(spanSlice, &s)
	}

	result := &v1_trace.TracesData{
		ResourceSpans: []*v1_trace.ResourceSpans{
			{
				Resource: nil,
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Scope:     nil,
						Spans:     spanSlice,
						SchemaUrl: "",
					},
				},
				SchemaUrl: "",
			},
		},
	}
	return result
}

func convertAttributes(attr map[string]string) []*v1_common.KeyValue {
	var result []*v1_common.KeyValue
	for key, value := range attr {
		result = append(result, &v1_common.KeyValue{
			Key:   key,
			Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: value}},
		})
	}
	return result
}

func convertEvents(evnetNames []string, eventTps []time.Time, eventAttributes []map[string]string) []*v1_trace.Span_Event {
	var result []*v1_trace.Span_Event
	for index := range evnetNames {
		result = append(result, &v1_trace.Span_Event{
			TimeUnixNano: uint64(eventTps[index].UnixNano()),
			Name:         evnetNames[index],
			Attributes:   convertAttributes(eventAttributes[index]),
		})
	}
	return result
}

func convertLinks(linkTraceId []string, linkSpanIds []string, linkTraceStates []string, linkAttributes []map[string]string) []*v1_trace.Span_Link {
	var result []*v1_trace.Span_Link
	for index := range linkTraceId {
		result = append(result, &v1_trace.Span_Link{
			TraceId:    []byte(linkTraceId[index]),
			SpanId:     []byte(linkSpanIds[index]),
			TraceState: linkTraceStates[index],
			Attributes: convertAttributes(linkAttributes[index]),
		})
	}
	return result
}
