package plugin

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/clickhouse"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/es"
)

type StorageConfig struct {
	StorageType       string                     `mapstructure:"storage_type"`
	ElasticsearchType *es.ElasticsearchType      `mapstructure:"elasticsearch"`
	ClickhouseType    *clickhouse.ClickhouseType `mapstructure:"clickhouse"`
}
