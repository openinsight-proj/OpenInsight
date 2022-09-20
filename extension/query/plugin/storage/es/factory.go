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

type ElasticsearchType struct {
	TracesIndex string   `mapstructure:"traces_index"`
	Endpoints   []string `mapstructure:"endpoints"`
	// User is used to configure HTTP Basic Authentication.
	User string `mapstructure:"user"`

	// Password is used to configure HTTP Basic Authentication.
	Password string `mapstructure:"password"`
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	logger *zap.Logger

	client *client.Elastic
	cfg    *ElasticsearchType
}

func (f *Factory) Initialize(logger *zap.Logger) error {

	logger.Info("init elasticsearch storage factory...")
	c, err := client.New(f.cfg.Endpoints, f.cfg.User, f.cfg.Password, f.cfg.TracesIndex)
	if err != nil {
		return err
	}
	f.client = c
	return nil
}

func (f *Factory) CreateSpanQuery() (storage.Query, error) {
	return &ElasticsearchQuery{
		client: f.client,
	}, nil
}

// Close closes the resources held by the factory
func (f *Factory) Close() error {
	return nil
}

// NewFactory creates a new Factory.
func NewFactory(es *ElasticsearchType) *Factory {
	return &Factory{
		cfg: es,
	}
}