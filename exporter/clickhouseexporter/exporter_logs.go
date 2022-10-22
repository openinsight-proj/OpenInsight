// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouseexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // For register database driver.
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"
)

type logsExporter struct {
	client    *sql.DB
	insertSQL string

	logger *zap.Logger
	cfg    *Config
}

func newLogsExporter(logger *zap.Logger, cfg *Config) (*logsExporter, error) {

	if err := createDatabase(cfg); err != nil {
		return nil, err
	}

	client, err := newClickhouseClient(cfg)
	if err != nil {
		return nil, err
	}

	if err = createLogsTable(cfg, client); err != nil {
		return nil, err
	}

	return &logsExporter{
		client:    client,
		insertSQL: renderInsertLogsSQL(cfg),
		logger:    logger,
		cfg:       cfg,
	}, nil
}

// Shutdown will shutdown the exporter.
func (e *logsExporter) Shutdown(_ context.Context) error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

func (e *logsExporter) pushLogsData(ctx context.Context, ld plog.Logs) error {
	start := time.Now()
	err := doWithTx(ctx, e.client, func(tx *sql.Tx) error {
		statement, err := tx.PrepareContext(ctx, e.insertSQL)
		if err != nil {
			return fmt.Errorf("PrepareContext:%w", err)
		}
		defer func() {
			_ = statement.Close()
		}()
		var serviceName string
		for i := 0; i < ld.ResourceLogs().Len(); i++ {
			logs := ld.ResourceLogs().At(i)
			res := logs.Resource()
			resAttr := attributesToMap(res.Attributes())
			if v, ok := res.Attributes().Get(conventions.AttributeServiceName); ok {
				serviceName = v.Str()
			}
			for j := 0; j < logs.ScopeLogs().Len(); j++ {
				rs := logs.ScopeLogs().At(j).LogRecords()
				for k := 0; k < rs.Len(); k++ {
					r := rs.At(k)
					logAttr := attributesToMap(r.Attributes())
					_, err = statement.ExecContext(ctx,
						r.Timestamp().AsTime(),
						r.TraceID().HexString(),
						r.SpanID().HexString(),
						uint32(r.Flags()),
						r.SeverityText(),
						int32(r.SeverityNumber()),
						serviceName,
						r.Body().AsString(),
						resAttr,
						logAttr,
					)
					if err != nil {
						return fmt.Errorf("ExecContext:%w", err)
					}
				}
			}
		}
		return nil
	})
	duration := time.Since(start)
	e.logger.Info("insert logs", zap.Int("records", ld.LogRecordCount()),
		zap.String("cost", duration.String()))
	return err
}

func attributesToMap(attributes pcommon.Map) map[string]string {
	m := make(map[string]string, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		m[k] = v.Str()
		return true
	})
	return m
}

const (
	// language=ClickHouse SQL
	createLogsTableSQL = `
CREATE TABLE IF NOT EXISTS %s (
     Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
     TraceId String CODEC(ZSTD(1)),
     SpanId String CODEC(ZSTD(1)),
     TraceFlags UInt32 CODEC(ZSTD(1)),
     SeverityText LowCardinality(String) CODEC(ZSTD(1)),
     SeverityNumber Int32 CODEC(ZSTD(1)),
     ServiceName LowCardinality(String) CODEC(ZSTD(1)),
     Body String CODEC(ZSTD(1)),
     ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     LogAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
     INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_log_attr_key mapKeys(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_log_attr_value mapValues(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
) ENGINE MergeTree()
%s
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SeverityText, toUnixTimestamp(Timestamp), TraceId)
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	insertLogsSQLTemplate = `INSERT INTO %s (
                        Timestamp,
                        TraceId,
                        SpanId,
                        TraceFlags,
                        SeverityText,
                        SeverityNumber,
                        ServiceName,
                        Body,
                        ResourceAttributes,
                        LogAttributes
                        ) VALUES (
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?,
                                  ?
                                  )`
)

var driverName = "clickhouse" // for testing

// newClickhouseClient create a clickhouse client.
func newClickhouseClient(cfg *Config) (*sql.DB, error) {
	return sql.Open(driverName, cfg.DSN)
}

func createDatabase(cfg *Config) error {
	database, _ := parseDSNDatabase(cfg.DSN)
	if database == defaultDatabase {
		return nil
	}
	// use default database to create new database
	dsnUseDefaultDatabase := strings.Replace(cfg.DSN, database, defaultDatabase, 1)
	db, err := sql.Open(driverName, dsnUseDefaultDatabase)
	if err != nil {
		return fmt.Errorf("sql.Open:%w", err)
	}
	defer func() {
		_ = db.Close()
	}()
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", database)
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("create database:%w", err)
	}
	return nil
}

func createLogsTable(cfg *Config, db *sql.DB) error {
	if _, err := db.Exec(renderCreateLogsTableSQL(cfg)); err != nil {
		return fmt.Errorf("exec create logs table sql: %w", err)
	}
	return nil
}

func renderCreateLogsTableSQL(cfg *Config) string {
	var ttlExpr string
	if cfg.TTLDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(Timestamp) + toIntervalDay(%d)`, cfg.TTLDays)
	}
	return fmt.Sprintf(createLogsTableSQL, cfg.LogsTableName, ttlExpr)
}

func renderInsertLogsSQL(cfg *Config) string {
	return fmt.Sprintf(insertLogsSQLTemplate, cfg.LogsTableName)
}

func doWithTx(_ context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
