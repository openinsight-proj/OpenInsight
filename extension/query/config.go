package query

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Protocols struct {
	Grpc *configgrpc.GRPCServerSettings `mapstructure:"grpc"`
	Http *confighttp.HTTPServerSettings `mapstructure:"http"`
}

type Config struct {
	config.ExtensionSettings `mapstructure:",squash"`
	Protocols                `mapstructure:"protocols"`
	TracingQuery             *plugin.StorageConfig `mapstructure:"tracing_query"`
	MetricsQuery             *plugin.StorageConfig `mapstructure:"metrics_query"`
	LoggingQuery             *plugin.StorageConfig `mapstructure:"logging_query"`
}
