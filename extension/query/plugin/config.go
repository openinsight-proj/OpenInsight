package plugin

type StorageConfig struct {
	StorageType       string             `mapstructure:"storage_type"`
	ElasticsearchType *ElasticsearchType `mapstructure:"elasticsearch"`
	ClickhouseType    *ClickhouseType    `mapstructure:"clickhouse"`
}

type ElasticsearchType struct {
	TracesIndex string   `mapstructure:"traces_index"`
	Endpoints   []string `mapstructure:"endpoints"`
	// User is used to configure HTTP Basic Authentication.
	User string `mapstructure:"user"`

	// Password is used to configure HTTP Basic Authentication.
	Password string `mapstructure:"password"`
}

type ClickhouseType struct {
	Dsn     string `mapstructure:"dsn"`
	Ttl     int64  `mapstructure:"ttl_days"`
	Timeout string `mapstructure:"timeout"`
}
