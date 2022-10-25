package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configtls"
	"go.uber.org/zap"
)

var ct = ClickhouseType{
	Dsn: "tcp://10.6.229.191:32022/otel",
	TlsClientSetting: configtls.TLSClientSetting{
		TLSSetting:         configtls.TLSSetting{},
		Insecure:           true,
		InsecureSkipVerify: false,
		ServerName:         "",
	},
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
