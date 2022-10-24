package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_common "go.opentelemetry.io/proto/otlp/common/v1"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_resource "go.opentelemetry.io/proto/otlp/resource/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
)

const (
	SUB_SQL = "SELECT TraceId AS id FROM %s_trace_id_ts %s ORDER BY Start DESC %s"
	COLUMNS = `a.Timestamp,
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
       a.Links.Attributes`
	BASE_SQL = `SELECT %s FROM
       (%s) AS b JOIN
       %s AS a on b.id = a.TraceId %s`
	DATETIME_LAYOUT          = "2006-01-02 15:04:05"
	QUERY_SERVICE_SQL        = "select ServiceName from (SELECT ServiceName,Timestamp FROM %s where Timestamp between date_sub(%s,%d,now()) and now()) group by ServiceName"
	QUERY_SERVICE_TIME_UNIT  = "DAY"
	QUERY_SERVICE_TIME_VALUE = 1
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

func (q *ClickHouseQuery) GetService(ctx context.Context) ([]string, error) {
	sql := fmt.Sprintf(QUERY_SERVICE_SQL, q.tracingTableName, QUERY_SERVICE_TIME_UNIT, QUERY_SERVICE_TIME_VALUE)
	var serviceList []string
	rows, err := q.client.Query(ctx, sql)
	if err != nil {
		return nil, err
	}

	var result struct {
		ServiceName string `ch:"ServiceName"`
	}
	for {
		if !rows.Next() {
			break
		}
		err = rows.ScanStruct(&result)
		if err != nil {
			return nil, err
		}
		serviceList = append(serviceList, result.ServiceName)
	}

	return serviceList, nil
}

func (q *ClickHouseQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	if traceID == "" {
		return nil, errors.New("traceID must not empty")
	}

	sql := fmt.Sprintf("SELECT %s FROM %s AS a WHERE a.TraceId='%s'", COLUMNS, q.tracingTableName, traceID)
	var result []TracesModel
	if err := q.client.Select(ctx, &result, sql); err != nil {
		return nil, err
	}

	return convertSpan(result), nil
}

func (q *ClickHouseQuery) SearchTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {
	sql, err := buildQuery(query, q.tracingTableName)
	if err != nil {
		return nil, err
	}

	var result []TracesModel
	if err = q.client.Select(ctx, &result, sql); err != nil {
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
	if query.ServiceName == "" {
		return "", errors.New("query parameter must contain  ServiceName")
	}

	var WherekeywordList []string
	WherekeywordList = append(WherekeywordList, fmt.Sprintf("a.ServiceName='%s'", query.ServiceName))
	if query.OperationName != "" {
		WherekeywordList = append(WherekeywordList, fmt.Sprintf("a.SpanName='%s'", query.OperationName))
	}
	if query.Tags != nil {
		for key, value := range query.Tags {
			WherekeywordList = append(WherekeywordList, fmt.Sprintf("a.SpanAttributes['%s']='%s'", key, value))
		}
	}
	whereKeywordCondition := fmt.Sprintf("WHERE %s", strings.Join(WherekeywordList, " AND "))

	// build time and LIMIT condition
	var whereTimeList []string

	//default past 1 hours
	end := time.Now()
	start := end.Add(-time.Hour)
	timePattern := "Start BETWEEN '%s' AND '%s'"
	timeCondition := fmt.Sprintf(timePattern, start.Format(DATETIME_LAYOUT), end.Format(DATETIME_LAYOUT))
	if query.EndTime.After(query.StartTime) {
		timeCondition = fmt.Sprintf(timePattern,
			query.StartTime.Format(DATETIME_LAYOUT),
			query.EndTime.Format(DATETIME_LAYOUT))
	}
	whereTimeList = append(whereTimeList, timeCondition)

	//DurationMin <= a.Duration <= DurationMax
	if query.DurationMin != nil && query.DurationMax != nil && query.DurationMin.Nanos < query.DurationMax.Nanos {
		whereTimeList = append(whereTimeList, fmt.Sprintf("(End - Start) BETWEEN %d AND %d", query.DurationMin.Nanos,
			query.DurationMax.Nanos))
	}
	limitPattern := "LIMIT %d"
	limitCondition := fmt.Sprintf(limitPattern, 100)
	if query.NumTraces > 0 && query.NumTraces < 100 {
		limitCondition = fmt.Sprintf(limitPattern, query.NumTraces)
	}
	whereTimeCondition := fmt.Sprintf("WHERE %s", strings.Join(whereTimeList, " AND "))

	subQuery := fmt.Sprintf(SUB_SQL, tableName, whereTimeCondition, limitCondition)
	return fmt.Sprintf(BASE_SQL, COLUMNS, subQuery, tableName, whereKeywordCondition), nil
}

func convertSpan(tracesModel []TracesModel) *v1_trace.TracesData {
	rsMap := make(map[string]*v1_trace.ResourceSpans)
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

		attrId := generateAttributesId(item.ResourceAttributes)
		if _, ok := rsMap[attrId]; ok {
			rsMap[attrId].ScopeSpans[0].Spans = append(rsMap[attrId].ScopeSpans[0].Spans, &s)
		} else {
			spanSlice := []*v1_trace.Span{&s}
			rsMap[attrId] = &v1_trace.ResourceSpans{
				Resource:   &v1_resource.Resource{Attributes: convertAttributes(item.ResourceAttributes)},
				ScopeSpans: []*v1_trace.ScopeSpans{{Spans: spanSlice}},
			}
		}
	}

	var rsList []*v1_trace.ResourceSpans
	for _, value := range rsMap {
		rsList = append(rsList, value)
	}

	return &v1_trace.TracesData{
		ResourceSpans: rsList,
	}
}

func generateAttributesId(attr map[string]string) string {
	var attrList []string
	for key, value := range attr {
		attrList = append(attrList, fmt.Sprintf("%s:%s", key, value))
	}
	sort.Slice(attrList, func(i, j int) bool { return attrList[i] <= attrList[j] })
	return strings.Join(attrList, ";")
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
