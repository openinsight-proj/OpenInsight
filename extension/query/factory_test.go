package query

import (
	"context"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestTypeStr(t *testing.T) {
	factory := NewFactory()

	assert.Equal(t, "query", string(factory.Type()))
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configtest.CheckConfigStruct(cfg))
}

func TestFactory_CreateExtension(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Endpoint:  defaultGRPCBindEndpoint,
			Transport: "tcp",
		},
	}

	ext, err := createExtension(context.Background(), componenttest.NewNopExtensionCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
}
