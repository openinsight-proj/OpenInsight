package clickhouse

import (
	"context"
	"io"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.opentelemetry.io/collector/config/configtls"
	"go.uber.org/zap"
)

var (
	_ io.Closer = (*Factory)(nil)
)

type ClickhouseType struct {
	Dsn              string                     `mapstructure:"dsn"`
	TlsClientSetting configtls.TLSClientSetting `mapstructure:"tls_setting"`
	LoggingTableName string                     `mapstructure:"logging_table_name"`
	TracingTableName string                     `mapstructure:"tracing_table_name"`
	MetricsTableName string                     `mapstructure:"metrics_table_name"`
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	logger *zap.Logger

	client clickhouse.Conn
	cfg    *ClickhouseType
}

func (f *Factory) Initialize(logger *zap.Logger) error {
	options, err := clickhouse.ParseDSN(f.cfg.Dsn)
	if err != nil {
		return err
	}

	if f.cfg.TlsClientSetting.ServerName == "" {
		f.cfg.TlsClientSetting.ServerName = "<missing service name>"
	}
	tls, err := f.cfg.TlsClientSetting.LoadTLSConfig()
	if err != nil {
		return err
	}
	options.TLS = tls

	conn, err := clickhouse.Open(options)
	if err != nil {
		return err
	}
	if err = conn.Ping(context.Background()); err != nil {
		return err
	}

	f.client = conn
	f.logger = logger
	return nil
}

func (f *Factory) CreateSpanQuery() (storage.Query, error) {
	return &ClickHouseQuery{
		logger:           f.logger,
		client:           f.client,
		loggingTableName: f.cfg.LoggingTableName,
		tracingTableName: f.cfg.TracingTableName,
		metricsTableName: f.cfg.MetricsTableName,
	}, nil
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
