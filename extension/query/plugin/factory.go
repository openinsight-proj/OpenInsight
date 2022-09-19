package plugin

import (
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/es"
	"go.uber.org/zap"
)

const (
	elasticsearchQueryType = "elasticsearch"
	prometheusQueryType    = "prometheus"
)

// AllQueryTypes defines all available storage backends
var AllStorageTypes = []string{
	elasticsearchQueryType,
	prometheusQueryType,
}

type FactoryConfig struct {
	TracingQueryType *StorageConfig
	LoggingQueryType *StorageConfig
	MetricsQueryType *StorageConfig
}

type Factory struct {
	FactoryConfig
	factories map[string]storage.Factory
	sConfig   *query.Config
}

func (f *Factory) getFactoryOfType(factoryType string) (storage.Factory, error) {
	switch factoryType {
	case elasticsearchQueryType:
		return es.NewFactory(), nil
	case prometheusQueryType:
		return es.NewFactory(), nil
	default:
		return nil, fmt.Errorf("unknown query type %s. Valid types are %v", factoryType, AllStorageTypes)
	}
}

// NewFactory creates the meta-factory.
func NewFactory(sConfig *query.Config, config FactoryConfig) (*Factory, error) {
	f := &Factory{FactoryConfig: config, sConfig: sConfig}
	uniqueTypes := []string{f.TracingQueryType.StorageType, f.LoggingQueryType.StorageType, f.MetricsQueryType.StorageType}
	f.factories = make(map[string]storage.Factory)

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
func (f *Factory) Initialize(cfg *query.Config, logger *zap.Logger) error {
	for _, factory := range f.factories {
		if err := factory.Initialize(logger); err != nil {
			return err
		}
	}

	return nil
}

func (f *Factory) CreateSpanQuery() (storage.Query, error) {
	factory, ok := f.factories[f.TracingQueryType.StorageType]
	if !ok {
		return nil, fmt.Errorf("no %s backend registered for span store", f.TracingQueryType)
	}
	return factory.CreateSpanQuery()
}
