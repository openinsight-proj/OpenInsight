package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aquasecurity/esquery"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource"
	v1_common "go.opentelemetry.io/proto/otlp/common/v1"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_resource "go.opentelemetry.io/proto/otlp/resource/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"strings"
	"time"
)

var (
	errParsTime = fmt.Errorf("start time must before endtime")
)

const (
	DATE_LAYOUT = "2006-01-02T15:04:05.000000000Z"
)

type ElasticsearchQuery struct {
	client       *client.Elastic
	SpanIndex    string
	LoggingIndex string
	MetricsIndex string
}

func (q *ElasticsearchQuery) GetService(ctx context.Context) ([]*v1_resource.Resource, error) {

	// boolean search query
	query := esquery.Search()
	query.Aggs(
		esquery.TermsAgg("service_name_aggregation", "Resource.service.name.keyword").Order(map[string]string{"_count": "desc"}).Size(100),
	).Size(0)

	res, err := q.client.DoSearch(ctx, q.SpanIndex, query)
	if err != nil {
		return nil, err
	}

	var services []*v1_resource.Resource
	for _, agg := range res.Aggregations {
		rMaps, err := DecodeSearchResult(*agg)
		if err != nil {
			return nil, err
		}
		for k, v := range rMaps {
			switch k {
			case "buckets":
				values := v.([]interface{})
				for _, value := range values {
					nameV := value.(map[string]interface{})
					name := nameV["key"]
					services = append(services, &v1_resource.Resource{
						Attributes: []*v1_common.KeyValue{
							{
								Key:   "service.name",
								Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: name.(string)}},
							},
						},
					})
				}
			}
		}
	}

	return services, nil
}

func (q *ElasticsearchQuery) SearchTraces(ctx context.Context, query *datasource.TraceQueryParameters) (*v1alpha1.TracesData, error) {
	ids, err := q.FindTraceIds(ctx, query)
	if err != nil {
		return nil, err
	}

	traces, err := q.MultiGetTraces(ctx, ids...)
	if err != nil {
		zap.S().Errorf("failed to query traces:%v", err)
	}
	return traces, nil
}

func (q *ElasticsearchQuery) MultiGetTraces(ctx context.Context, traceIds ...string) (*v1alpha1.TracesData, error) {
	qe := esquery.Search()
	boolQ := esquery.Bool()
	boolQ.Must(Terms("TraceId", traceIds...))
	// TODO(jian): 65536(es default limit) right?
	qe.Query(boolQ).Size(5000)
	res, err := q.client.DoSearch(ctx, q.SpanIndex, qe)
	if err != nil {
		return nil, err
	}
	tracesData, err := DocumentsResourceSpansConvert(res.Hits)
	if err != nil {
		return nil, err
	}

	return datasource.DocumentsTracesConvert(tracesData)
}

func (q *ElasticsearchQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	qe := esquery.Search()
	boolQ := esquery.Bool()
	boolQ.Must(esquery.Term("TraceId", traceID))
	qe.Query(boolQ)

	res, err := q.client.DoSearch(ctx, q.SpanIndex, qe)
	if err != nil {
		return nil, err
	}
	return DocumentsResourceSpansConvert(res.Hits)
}

func (q *ElasticsearchQuery) SearchLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) GetOperations(ctx context.Context, params *datasource.OperationsQueryParameters) ([]string, error) {
	// boolean search query
	query := esquery.Search()
	boolQ := esquery.Bool()

	if params.ServiceName != "" {
		boolQ.Must(esquery.Term("Resource.service.name", params.ServiceName))
	}
	if params.SpanKind != "" {
		boolQ.Must(esquery.Term("Kind.keyword", params.SpanKind))
	} else {
		// query CLIENT/SEVER as ingress/egress default
		boolQ.MustNot(esquery.Term("Kind.keyword", "SPAN_KIND_INTERNAL"))
	}

	query.Query(boolQ)
	query.Aggs(
		esquery.TermsAgg("service_operations", "Name.keyword").Order(map[string]string{"_count": "desc"}).Size(10000),
	).Size(0)

	res, err := q.client.DoSearch(ctx, q.SpanIndex, query)
	if err != nil {
		return nil, err
	}

	var operations []string
	for _, agg := range res.Aggregations {
		rMaps, err := DecodeSearchResult(*agg)
		if err != nil {
			return nil, err
		}
		for k, v := range rMaps {
			switch k {
			case "buckets":
				values := v.([]interface{})
				for _, value := range values {
					nameV := value.(map[string]interface{})
					name := nameV["key"]
					operations = append(operations, name.(string))
				}
			}
		}
	}

	return operations, nil
}

func (q *ElasticsearchQuery) FindTraceIds(ctx context.Context, queryParams *datasource.TraceQueryParameters) ([]string, error) {
	idsQsl, err := buildTraceIdsQuery(queryParams)
	if err != nil {
		return nil, err
	}

	res, err := q.client.DoSearch(ctx, q.SpanIndex, idsQsl)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, agg := range res.Aggregations {
		rMaps, err := DecodeSearchResult(*agg)
		if err != nil {
			return nil, err
		}
		for k, v := range rMaps {
			switch k {
			case "buckets":
				values := v.([]interface{})
				for _, value := range values {
					idV := value.(map[string]interface{})
					id := idV["key"]
					ids = append(ids, id.(string))
				}
			}
		}
	}
	return ids, nil
}

func buildTraceIdsQuery(params *datasource.TraceQueryParameters) (*esquery.SearchRequest, error) {
	q, err := buildTraceQuery(params)
	if err != nil {
		return nil, err
	}

	aggs := esquery.TermsAgg("traceIDs", "TraceId.keyword").Order(map[string]string{"_count": "desc"})
	if params.NumTraces > 0 {
		aggs.Size(uint64(params.NumTraces))
	} else {
		aggs.Size(uint64(20))
	}

	return q.Aggs(aggs), nil
}

// Build the request body.
func buildTraceQuery(params *datasource.TraceQueryParameters) (*esquery.SearchRequest, error) {
	// boolean search query
	q := esquery.Search()
	boolQ := esquery.Bool()
	if params.ServiceName != "" {
		boolQ.Must(esquery.Term("Resource.service.name.keyword", params.ServiceName))
	}
	if params.OperationName != "" {
		boolQ.Must(esquery.Term("Name.keyword", params.OperationName))
	}

	if !params.StartTime.IsZero() && params.EndTime.IsZero() {
		boolQ.Must(esquery.Range("@timestamp").Gte(params.StartTime.Format(DATE_LAYOUT)))
	}

	if params.StartTime.IsZero() && !params.EndTime.IsZero() {
		boolQ.Must(esquery.Range("@timestamp").Lte(params.EndTime.Format(DATE_LAYOUT)))
	}

	if !params.StartTime.IsZero() && !params.EndTime.IsZero() && params.StartTime.Before(params.EndTime) {
		boolQ.Must(esquery.Range("@timestamp").Gte(params.StartTime.Format(DATE_LAYOUT)).Lte(params.EndTime.Format(DATE_LAYOUT)))
	} else {
		return q, errParsTime
	}

	if len(params.Tags) > 0 {
		for k, v := range params.Tags {
			boolQ.Must(esquery.Term(k, v))
		}
	}

	q.Query(boolQ)
	return q, nil
}

func DecodeSearchResult(jsonRaw json.RawMessage) (map[string]interface{}, error) {
	rMaps := make(map[string]interface{})
	d := json.NewDecoder(bytes.NewReader(jsonRaw))
	d.UseNumber()
	if err := d.Decode(&rMaps); err != nil {
		typeErr := err.(*json.UnmarshalTypeError)
		zap.S().Errorf("failed to decode  searchHits %s and typeErr: %s", zap.Error(err).String, typeErr.Field)
		return nil, err
	}
	return rMaps, nil
}

func DocumentsResourceSpansConvert(searchHits *client.SearchHits) (*v1_trace.TracesData, error) {
	rSpans := make([]*v1_trace.ResourceSpans, len(searchHits.Hits))

	for i, hit := range searchHits.Hits {
		rSpansMaps, err := DecodeSearchResult(*hit.Source)
		if err != nil {
			return nil, err
		}

		span := v1_trace.Span{}
		resource := v1_resource.Resource{}
		var sAttributes []*v1_common.KeyValue
		var rAttributes []*v1_common.KeyValue
		for k, v := range rSpansMaps {
			switch k {
			case "TraceId":
				span.TraceId = []byte(v.(string))
			case "TraceStatus":
				statusCode, _ := v.(json.Number).Int64()
				span.Status = &v1_trace.Status{Code: v1_trace.Status_StatusCode(statusCode)}
				span.TraceState = v1_trace.Status_StatusCode_name[int32(statusCode)]
			case "Name":
				span.Name = v.(string)
			case "EndTimestamp":
				t, err := time.Parse(DATE_LAYOUT, v.(string))
				if err != nil {
					zap.S().Errorf("failed to parse endtimestamp %s", zap.Error(err).String)
				}
				span.EndTimeUnixNano = uint64(t.UnixNano())
			case "@timestamp":
				t, err := time.Parse(DATE_LAYOUT, v.(string))
				if err != nil {
					zap.S().Errorf("failed to parse @timestamp %s", zap.Error(err).String)
				}
				span.StartTimeUnixNano = uint64(t.UnixNano())
			case "ParentSpanId":
				span.ParentSpanId = []byte(v.(string))
			case "SpanId":
				span.SpanId = []byte(v.(string))
			case "Kind":
				span.Kind = v1_trace.Span_SpanKind(v1_trace.Span_SpanKind_value[v.(string)])
			}

			if strings.Contains(k, "Attributes.") {
				sAttributes = append(sAttributes, &v1_common.KeyValue{
					Key:   strings.TrimPrefix(k, "Attributes."),
					Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprint(v)}},
				})
			}

			if strings.Contains(k, "Resource.") {
				rAttributes = append(rAttributes, &v1_common.KeyValue{
					Key:   strings.TrimPrefix(k, "Resource."),
					Value: &v1_common.AnyValue{Value: &v1_common.AnyValue_StringValue{StringValue: fmt.Sprint(v)}},
				})
			}
		}

		span.Attributes = sAttributes
		resource.Attributes = rAttributes

		rSpans[i] = &v1_trace.ResourceSpans{
			Resource: &resource,
			ScopeSpans: []*v1_trace.ScopeSpans{
				{
					Spans: []*v1_trace.Span{&span},
				},
			},
		}
	}
	return &v1_trace.TracesData{ResourceSpans: rSpans}, nil
}

type TermsQuery struct {
	field  string
	values []string
}

func (q *TermsQuery) Map() map[string]interface{} {
	return map[string]interface{}{
		"terms": map[string]interface{}{
			q.field: q.values,
		},
	}
}

func Terms(field string, values ...string) *TermsQuery {
	return &TermsQuery{
		field:  field,
		values: values,
	}
}
