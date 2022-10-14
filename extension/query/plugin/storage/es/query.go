package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aquasecurity/esquery"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
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

func (q *ElasticsearchQuery) FindTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {

	qsl := buildQuery(query)
	res, err := q.client.DoSearch(ctx, q.SpanIndex, qsl)
	if err != nil {
		return nil, err
	}

	return q.documentsConvert(res.Hits)
}

func (q *ElasticsearchQuery) FindLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

// Build the request body.
func buildQuery(params *storage.TraceQueryParameters) *esquery.SearchRequest {
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
	q.Query(boolQ)
	if params.NumTraces > 0 {
		q.Size(uint64(params.NumTraces))
	} else {
		q.Size(uint64(20))
	}

	return q
}

func (q *ElasticsearchQuery) documentsConvert(searchHits *client.SearchHits) (*v1_trace.TracesData, error) {
	traceData := &v1_trace.TracesData{}
	spans := make([]*v1_trace.Span, len(searchHits.Hits))
	for i, hit := range searchHits.Hits {
		spanMaps := make(map[string]interface{})
		d := json.NewDecoder(bytes.NewReader(*hit.Source))
		d.UseNumber()
		if err := d.Decode(&spanMaps); err != nil {
			typeErr := err.(*json.UnmarshalTypeError)
			fmt.Print(typeErr.Field)
			return nil, err
		}

		span := v1_trace.Span{}
		for k, v := range spanMaps {
			switch k {
			//TODO: make more sense
			case "Name":
				span.Name = v.(string)
			}
		}
		spans[i] = &span

		rs := []*v1_trace.ResourceSpans{
			{
				Resource: nil,
				ScopeSpans: []*v1_trace.ScopeSpans{
					{
						Spans: spans,
					},
				},
				SchemaUrl: "",
			},
		}
		traceData.ResourceSpans = rs
	}
	return traceData, nil
}
