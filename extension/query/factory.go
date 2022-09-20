package query

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

const (
	typeStr = "query"

	// Protocol values.
	protoGRPC = "Grpc"
	protoHTTP = "http"

	// Default endpoints to bind to.
	defaultGRPCBindEndpoint = "0.0.0.0:9090"
	defaultHTTPBindEndpoint = "0.0.0.0:8080"
)

// NewFactory creates a factory for the otlp query extension.
func NewFactory() component.ExtensionFactory {
	return component.NewExtensionFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
		component.StabilityLevelBeta,
	)
}

func createDefaultConfig() config.Extension {
	return &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewComponentID(typeStr)),
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
		TracingQuery: &plugin.StorageConfig{},
		LoggingQuery: &plugin.StorageConfig{},
		MetricsQuery: &plugin.StorageConfig{},
	}
}

func createExtension(_ context.Context, set component.ExtensionCreateSettings, cfg config.Extension) (component.Extension, error) {
	c := cfg.(*Config)

	return NewQueryServer(c, set.TelemetrySettings), nil
}
