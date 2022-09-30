package storage

import (
	"context"
	"github.com/golang/protobuf/ptypes/duration"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"time"
)

type Query interface {
	GetTrace(ctx context.Context, traceID string) (ptrace.Span, error)
	FindTraces(ctx context.Context, query *TraceQueryParameters) ([]*ptrace.Span, error)
	FindLogs(ctx context.Context) ([]*plog.Logs, error)
	GetLog(ctx context.Context) ([]*plog.LogRecord, error)

	//TODO: add metrics query.
}

// TraceQueryParameters contains parameters of a trace query.
type TraceQueryParameters struct {
	ServiceName   string
	OperationName string
	Tags          map[string]string
	StartTime     time.Time
	EndTime       time.Time
	DurationMin   *duration.Duration
	DurationMax   *duration.Duration
	NumTraces     int
}
