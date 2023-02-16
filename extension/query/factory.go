package query

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/extension"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/clickhouse"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/es"
)

const (
	typeStr = "query"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint = "0.0.0.0:9090"
	defaultHTTPBindEndpoint = "0.0.0.0:8080"
)

// NewFactory creates a factory for the otlp query extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
		component.StabilityLevelAlpha,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			Grpc: &configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  defaultGRPCBindEndpoint,
					Transport: "tcp",
				},
			},
			Http: &confighttp.HTTPServerSettings{
				Endpoint: defaultHTTPBindEndpoint,
			},
		},
		Storage: &Storage{
			ElasticsearchType: &es.ElasticsearchType{},
			ClickhouseType:    &clickhouse.ClickhouseType{},
		},
		TracingQuery: &plugin.StorageConfig{},
		LoggingQuery: &plugin.StorageConfig{},
		MetricsQuery: &plugin.StorageConfig{},
	}
}

func createExtension(_ context.Context, set extension.CreateSettings, cfg component.Config) (extension.Extension, error) {
	c := cfg.(*Config)
	return NewQueryServer(c, set.TelemetrySettings), nil
}
