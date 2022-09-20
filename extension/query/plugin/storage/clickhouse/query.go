package clickhouse

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ClickHouseQuery struct {
}

func (q *ClickHouseQuery) GetTrace(ctx context.Context, traceID string) (ptrace.Span, error) {
	return ptrace.Span{}, nil
}

func (q *ClickHouseQuery) FindTraces(ctx context.Context, query *storage.TraceQueryParameters) ([]*ptrace.Span, error) {
	return nil, nil
}

func (q *ClickHouseQuery) FindLogs(ctx context.Context) ([]*plog.Logs, error) {
	return nil, nil
}
func (q *ClickHouseQuery) GetLog(ctx context.Context) ([]*plog.LogRecord, error) {
	return nil, nil
}
