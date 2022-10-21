package clickhouse

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
)

type ClickHouseQuery struct {
}

func (q *ClickHouseQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	return &v1_trace.TracesData{}, nil
}

func (q *ClickHouseQuery) SearchTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {
	return nil, nil
}

func (q *ClickHouseQuery) SearchLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}
func (q *ClickHouseQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}
