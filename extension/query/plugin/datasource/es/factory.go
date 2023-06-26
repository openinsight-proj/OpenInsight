package es

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource"
	"go.uber.org/zap"
	"io"
)

var (
	_ io.Closer = (*Factory)(nil)
)

type ElasticsearchType struct {
	TracesIndex  string   `mapstructure:"traces_index"`
	LoggingIndex string   `mapstructure:"logs_index"`
	MetricsIndex string   `mapstructure:"metrics_index"`
	Endpoints    []string `mapstructure:"endpoints"`
	// User is used to configure HTTP Basic Authentication.
	User string `mapstructure:"user"`

	// Password is used to configure HTTP Basic Authentication.
	Password string `mapstructure:"password"`
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	client *client.Elastic
	cfg    *ElasticsearchType
}

func (f *Factory) Initialize(logger *zap.Logger) error {
	c, err := client.New(f.cfg.Endpoints, f.cfg.User, f.cfg.Password)
	if err != nil {
		logger.Error("initialize es client error")
		return err
	}
	f.client = c
	return nil
}

func (f *Factory) CreateSpanQuery() (datasource.Query, error) {
	return &ElasticsearchQuery{
		client:       f.client,
		SpanIndex:    f.cfg.TracesIndex,
		MetricsIndex: f.cfg.MetricsIndex,
		LoggingIndex: f.cfg.LoggingIndex,
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
