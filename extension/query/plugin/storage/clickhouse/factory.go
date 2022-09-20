package clickhouse

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.uber.org/zap"
	"io"
)

var (
	_ io.Closer = (*Factory)(nil)
)

type ClickhouseType struct {
	Dsn     string `mapstructure:"dsn"`
	Ttl     int64  `mapstructure:"ttl_days"`
	Timeout string `mapstructure:"timeout"`
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	logger *zap.Logger

	client *client.Elastic
	cfg    *ClickhouseType
}

func (f *Factory) Initialize(logger *zap.Logger) error {
	//TODO：
	return nil
}

func (f *Factory) CreateSpanQuery() (storage.Query, error) {
	return &ClickHouseQuery{}, nil
}

// Close closes the resources held by the factory
func (f *Factory) Close() error {
	return nil
}

// NewFactory creates a new Factory.
func NewFactory(ct *ClickhouseType) *Factory {
	return &Factory{
		cfg: ct,
	}
}
