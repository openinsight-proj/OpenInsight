package query

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/servicetest"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()

	factories.Extensions[typeStr] = factory
	cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", "test-config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Extensions), 2)
	defaultCfg := factory.CreateDefaultConfig()

	defaultCfg.(*Config).TracingQuery.StorageType = "elasticsearch"
	//defaultCfg.(*Config).TracingQuery.Endpoints = []string{"https://elastic.example.com:9200"}
	r0 := cfg.Extensions[config.NewComponentID(typeStr)]
	assert.Equal(t, r0, defaultCfg)
}

func TestLoadConfigSelector(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()

	factories.Extensions[typeStr] = factory
	cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", "test-config-2.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}
