package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.uber.org/zap"
)

var (
	_ io.Closer = (*Factory)(nil)
)

const (
	TIMEOUT = time.Second * 5
)

type ClickhouseType struct {
	Dsn              string `mapstructure:"dsn"`
	Ttl              int64  `mapstructure:"ttl_days"`
	Timeout          string `mapstructure:"timeout"`
	LoggingTableName string `mapstructure:"logging_table_name"`
	TracingTableName string `mapstructure:"tracing_table_name"`
	MetricsTableName string `mapstructure:"metrics_table_name"`
}

// Factory implements storage.Factory for Elasticsearch as storage.
type Factory struct {
	logger *zap.Logger

	client clickhouse.Conn
	cfg    *ClickhouseType
}

func (f *Factory) Initialize(logger *zap.Logger) error {
	//init client and logger
	dsnURL, _ := url.Parse(f.cfg.Dsn)
	options := &clickhouse.Options{
		Addr: []string{dsnURL.Host},
	}
	if dsnURL.Query().Get("database") == "" {
		return errors.New("dsn string must contain a database")
	}
	auth := clickhouse.Auth{
		Database: dsnURL.Query().Get("database"),
	}

	if dsnURL.Query().Get("username") != "" {
		auth.Username = dsnURL.Query().Get("username")
		auth.Password = dsnURL.Query().Get("password")
	}
	options.Auth = auth

	options.DialTimeout = TIMEOUT
	if f.cfg.Timeout != "" {
		timeout, err := time.ParseDuration(f.cfg.Timeout)
		if err == nil {
			options.DialTimeout = timeout
		}
		logger.Error(fmt.Sprintf("parse timeout string: %s error, use default timeout: %s", f.cfg.Timeout, TIMEOUT))
	}

	conn, err := clickhouse.Open(options)
	if err != nil {
		return err
	}
	if err = conn.Ping(context.Background()); err != nil {
		return err
	}
	v, _ := conn.ServerVersion()
	fmt.Println(v)
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
