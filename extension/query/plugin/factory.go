package plugin

import (
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource/clickhouse"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource/es"
	"go.uber.org/zap"
)

const (
	elasticsearchQueryType = "elasticsearch"
	prometheusQueryType    = "prometheus"
	clickhouseQueryType    = "clickhouse"
)

// AllQueryTypes defines all available storage backends
var AllStorageTypes = []string{
	elasticsearchQueryType,
	prometheusQueryType,
	clickhouseQueryType,
}

type FactoryConfig struct {
	ElasticsearchStorage *es.ElasticsearchType
	ClickhouseStorage    *clickhouse.ClickhouseType
	TracingQuery         *StorageConfig
	MetricsQuery         *StorageConfig
	LoggingQuery         *StorageConfig
	//TODO: add others
}
type Factory struct {
	factories map[string]datasource.Factory
	sConfig   *FactoryConfig
}

func (f *Factory) getFactoryOfType(factoryType string) (datasource.Factory, error) {
	switch factoryType {
	//TODO: refactoring
	case elasticsearchQueryType:
		return es.NewFactory(f.sConfig.ElasticsearchStorage), nil
	case clickhouseQueryType:
		return clickhouse.NewFactory(f.sConfig.ClickhouseStorage), nil
	default:
		return nil, fmt.Errorf("unknown query type %s. Valid types are %v", factoryType, AllStorageTypes)
	}
}

// NewFactory creates the meta-factory.
func NewFactory(cfg *FactoryConfig) (*Factory, error) {
	f := &Factory{sConfig: cfg}
	uniqueTypes := []string{}
	if cfg.TracingQuery.StorageType != "" {
		uniqueTypes = append(uniqueTypes, cfg.TracingQuery.StorageType)
	}
	if cfg.LoggingQuery.StorageType != "" {
		uniqueTypes = append(uniqueTypes, cfg.LoggingQuery.StorageType)
	}
	if cfg.MetricsQuery.StorageType != "" {
		uniqueTypes = append(uniqueTypes, cfg.MetricsQuery.StorageType)
	}
	f.factories = make(map[string]datasource.Factory)

	for _, t := range uniqueTypes {
		ff, err := f.getFactoryOfType(t)
		if err != nil {
			return nil, err
		}
		f.factories[t] = ff
	}
	return f, nil
}

// Initialize implements storage.Factory.
func (f *Factory) Initialize(logger *zap.Logger) error {
	for _, factory := range f.factories {
		if err := factory.Initialize(logger); err != nil {
			return err
		}
	}

	return nil
}

func (f *Factory) CreateSpanQuery() (datasource.Query, error) {
	factory, ok := f.factories[f.sConfig.TracingQuery.StorageType]
	if !ok {
		return nil, fmt.Errorf("no %s backend registered for span store", f.sConfig.TracingQuery.StorageType)
	}
	return factory.CreateSpanQuery()
}
