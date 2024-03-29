package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configtls"
)

var ct = ClickhouseType{
	Dsn: "tcp://127.0.0.1:9000/otel",
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

func TestCreateSpanQuery(t *testing.T) {
	factory := NewFactory(&ct)
	require.NotNil(t, factory)

	query, err := factory.CreateSpanQuery()
	require.NotNil(t, query)
	require.NoError(t, err)
}
