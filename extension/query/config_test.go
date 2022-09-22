package query

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/clickhouse"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/es"
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
	defaultCfg.(*Config).MetricsQuery.StorageType = "elasticsearch"
	defaultCfg.(*Config).LoggingQuery.StorageType = "elasticsearch"
	defaultCfg.(*Config).TracingQuery.ElasticsearchType = &es.ElasticsearchType{
		Endpoints:   []string{"http://localhost:9200"},
		User:        "elastic",
		Password:    "search",
		TracesIndex: "trace_index",
	}

	defaultCfg.(*Config).TracingQuery.ClickhouseType = &clickhouse.ClickhouseType{
		Dsn:     "tcp://127.0.0.1:9000?database=default",
		Ttl:     3,
		Timeout: "5s",
	}

	defaultCfg.(*Config).MetricsQuery.ElasticsearchType = &es.ElasticsearchType{
		Endpoints:   []string{"http://localhost:9200"},
		User:        "elastic",
		Password:    "search",
		TracesIndex: "trace_index",
	}

	defaultCfg.(*Config).LoggingQuery.ElasticsearchType = &es.ElasticsearchType{
		Endpoints:   []string{"http://localhost:9200"},
		User:        "elastic",
		Password:    "search",
		TracesIndex: "trace_index",
	}

	r0 := cfg.Extensions[config.NewComponentID(typeStr)]
	assert.Equal(t, r0, defaultCfg)
}
