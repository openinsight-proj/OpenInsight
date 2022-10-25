package query

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/service/servicetest"
	"path/filepath"
	"testing"
)

func Test_query_server(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()

	factories.Extensions[typeStr] = factory
	cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", "test-config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg_e := cfg.Extensions[config.NewComponentID(typeStr)]

	cfg_e.(*Config).Protocols.Grpc = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
	}

	cfg_e.(*Config).Protocols.Http = &confighttp.HTTPServerSettings{
		Endpoint: defaultHTTPBindEndpoint,
	}

	ext, err := createExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg_e)
	require.NoError(t, err)
	require.NotNil(t, ext)
	err = ext.Start(context.Background(), nil)
	if err != nil {
		return
	}
	select {}

}
