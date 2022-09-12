package es

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.uber.org/zap"
	"io"
)

var (
	_ io.Closer = (*Factory)(nil)
)

type Configuration struct {
	Servers  []string
	Username string
	Password string
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	logger *zap.Logger

	client *client.Elastic
	cfg    *Configuration
}

func (f *Factory) Initialize(logger *zap.Logger) error {

	logger.Info("init elasticsearch storage factory...")
	// TODO: init es client
	c, err := client.New("", "", "", "")
	if err != nil {
		return err
	}
	f.client = c
	return nil
}

func (f *Factory) CreateSpanQuery() (storage.Query, error) {
	return &ElasticsearchQuery{}, nil
}

// Close closes the resources held by the factory
func (f *Factory) Close() error {
	return nil
}

// NewFactory creates a new Factory.
func NewFactory() *Factory {
	return &Factory{}
}
