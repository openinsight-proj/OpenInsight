package es

import (
	"context"
	"github.com/aquasecurity/esquery"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"log"
)

type ElasticsearchQuery struct {
	client       *client.Elastic
	SpanIndex    string
	LoggingIndex string
	MetricsIndex string
}

type OtlpSpan struct {
}

func (q *ElasticsearchQuery) GetTrace(ctx context.Context, traceID string) (ptrace.Span, error) {
	return ptrace.Span{}, nil
}

func (q *ElasticsearchQuery) FindTraces(ctx context.Context, query *storage.TraceQueryParameters) ([]*ptrace.Span, error) {

	//TODO:
	qsl := buildQuery(query)
	res, err := q.client.DoSearch(ctx, q.SpanIndex, qsl)
	if err != nil {
		return nil, err
	}

	log.Printf("total hosts: %d", len(res.Hits.Hits))
	return nil, nil
}

func (q *ElasticsearchQuery) FindLogs(ctx context.Context) ([]*plog.Logs, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) GetLog(ctx context.Context) ([]*plog.LogRecord, error) {
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
