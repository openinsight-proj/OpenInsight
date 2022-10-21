package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aquasecurity/esquery"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
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

func (q *ElasticsearchQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	return &v1_trace.TracesData{}, nil
}

func (q *ElasticsearchQuery) SearchTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {

	qsl, err := buildQuery(query)
	res, err := q.client.DoSearch(ctx, q.SpanIndex, qsl)
	if err != nil {
		return nil, err
	}

	return DocumentsConvert(res.Hits)
}

func (q *ElasticsearchQuery) SearchLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

// Build the request body.
func buildQuery(params *storage.TraceQueryParameters) (*esquery.SearchRequest, error) {
	//{
	//  "size": 20,
	//  "query": {
	//    "bool": {
	//      "must": [
	//        {
	//          "range": {
	//            "@timestamp": {
	//              "gte": "2022-09-23T09:45:17.394924000Z",
	//              "lte": "2022-09-23T09:45:17.394924000Z"
	//            }
	//          }
	//        },
	//        {
	//          "term": {
	//            "Name": {
	//              "value": "VALUE"
	//            }
	//          }
	//        },
	//        {
	//          "term": {
	//            "Resource.service.name": {
	//              "value": "VALUE"
	//            }
	//          }
	//        }
	//      ]
	//    }
	//  }
	//}

	// boolean search query
	q := esquery.Search()
	boolQ := esquery.Bool()
	if params.ServiceName != "" {
		boolQ.Must(esquery.Term("Resource.service.name", params.ServiceName))
	}
	if params.OperationName != "" {
		boolQ.Must(esquery.Term("Name", params.OperationName))
	}

	if !params.StartTime.IsZero() && params.EndTime.IsZero() {
		boolQ.Must(esquery.Range("@timestamp").Gte(params.StartTime.UTC().String()))
	}

	if params.StartTime.IsZero() && !params.EndTime.IsZero() {
		boolQ.Must(esquery.Range("@timestamp").Lte(params.EndTime.UTC().String()))
	}

	if !params.StartTime.IsZero() && !params.EndTime.IsZero() && params.StartTime.Before(params.EndTime) {
		boolQ.Must(esquery.Range("@timestamp").Gte(params.StartTime.UTC().String()).Lte(params.EndTime.UTC().String()))
	} else {
		return q, errParsTime
	}

	if len(params.Tags) > 0 {
		for k, v := range params.Tags {
			boolQ.Must(esquery.Term(k, v))
		}
	}

	if params.DurationMin != nil {
		//TODO: do not support duration filters.
	}

	q.Query(boolQ)
	if params.NumTraces > 0 {
		q.Size(uint64(params.NumTraces))
	} else {
		q.Size(uint64(20))
	}

	return q, nil
}

func DocumentsConvert(searchHits *client.SearchHits) (*v1_trace.TracesData, error) {
	rSpans := make([]*v1_trace.ResourceSpans, len(searchHits.Hits))

	for i, hit := range searchHits.Hits {
		rSpansMaps := make(map[string]interface{})
		d := json.NewDecoder(bytes.NewReader(*hit.Source))
		d.UseNumber()
		if err := d.Decode(&rSpansMaps); err != nil {
			typeErr := err.(*json.UnmarshalTypeError)
			zap.S().Errorf("failed to decode  searchHits %s and typeErr: %s", zap.Error(err).String, typeErr.Field)
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
