package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var ct = ClickhouseType{
	Dsn:              "tcp://127.0.0.1:9000?database=otel",
	Ttl:              0,
	Timeout:          "",
	LoggingTableName: "otel_logs",
	TracingTableName: "otel_traces",
	MetricsTableName: "otel_metrics",
}

func TestInitialize(t *testing.T) {
	factory := NewFactory(&ct)
	require.NotNil(t, factory)

	err := factory.Initialize(&zap.Logger{})
	require.NoError(t, err)
}

func TestCreateSpanQuery(t *testing.T) {
	factory := NewFactory(&ct)
	require.NotNil(t, factory)

	query, err := factory.CreateSpanQuery()
	require.NotNil(t, query)
	require.NoError(t, err)
}
