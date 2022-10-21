package storage

import (
	"context"
	"github.com/golang/protobuf/ptypes/duration"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"time"
)

type Query interface {
	GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error)
	FindTraces(ctx context.Context, query *TraceQueryParameters) (*v1_trace.TracesData, error)
	FindLogs(ctx context.Context) (*v1_logs.LogsData, error)
	GetLog(ctx context.Context) (*v1_logs.LogsData, error)

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
