package query

import (
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
)

type Protocols struct {
	Grpc *configgrpc.GRPCServerSettings `mapstructure:"Grpc"`
	Http *confighttp.HTTPServerSettings `mapstructure:"http"`
}

type StorageConfig struct {
	storageType *string   `mapstructure:"storage_type"`
	endpoints   []*string `mapstructure:"endpoints"`
}

type Config struct {
	config.ExtensionSettings `mapstructure:",squash"`
	Protocols                `mapstructure:"protocols"`
	TracingQuery             StorageConfig `mapstructure:"tracing_query"`
	MetricsQuery             StorageConfig `mapstructure:"metrics_query"`
	LoggingQuery             StorageConfig `mapstructure:"logging_query"`
}
