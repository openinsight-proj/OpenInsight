package es

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ElasticsearchQuery struct {
	client *client.Elastic
}

func (q *ElasticsearchQuery) GetTrace(ctx context.Context, traceID string) (ptrace.Span, error) {
	return ptrace.Span{}, nil
}

func (q *ElasticsearchQuery) FindTraces(ctx context.Context, query *storage.TraceQueryParameters) ([]*ptrace.Span, error) {
	return nil, nil
}

func (q *ElasticsearchQuery) FindLogs(ctx context.Context) ([]*plog.Logs, error) {
	return nil, nil
}
func (q *ElasticsearchQuery) GetLog(ctx context.Context) ([]*plog.LogRecord, error) {
	return nil, nil
}
