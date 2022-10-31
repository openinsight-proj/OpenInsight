package query

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
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

	ext, err := createExtension(context.Background(),
		componenttest.NewNopExtensionCreateSettings(),
		cfg.Extensions[config.NewComponentID(typeStr)])
	require.NoError(t, err)
	require.NotNil(t, ext)
	err = ext.Start(context.Background(), nil)
	if err != nil {
		return
	}
	select {}
}
