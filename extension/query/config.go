package query

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/clickhouse"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/es"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Protocols struct {
	Grpc *configgrpc.GRPCServerSettings `mapstructure:"grpc"`
	Http *confighttp.HTTPServerSettings `mapstructure:"http"`
}

type Config struct {
	Protocols    `mapstructure:"protocols"`
	Storage      *Storage              `mapstructure:"storage"`
	TracingQuery *plugin.StorageConfig `mapstructure:"tracing_query"`
	MetricsQuery *plugin.StorageConfig `mapstructure:"metrics_query"`
	LoggingQuery *plugin.StorageConfig `mapstructure:"logging_query"`
}

type Storage struct {
	ElasticsearchType *es.ElasticsearchType      `mapstructure:"elasticsearch"`
	ClickhouseType    *clickhouse.ClickhouseType `mapstructure:"clickhouse"`
	//TODO: add more storage
}
