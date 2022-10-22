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
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"
)

type tracesExporter struct {
	client    *sql.DB
	insertSQL string

	logger *zap.Logger
	cfg    *Config
}

func newTracesExporter(logger *zap.Logger, cfg *Config) (*tracesExporter, error) {

	if err := createDatabase(cfg); err != nil {
		return nil, err
	}

	client, err := newClickhouseClient(cfg)
	if err != nil {
		return nil, err
	}

	if err = createTracesTable(cfg, client); err != nil {
		return nil, err
	}

	return &tracesExporter{
		client:    client,
		insertSQL: renderInsertTracesSQL(cfg),
		logger:    logger,
		cfg:       cfg,
	}, nil
}

// Shutdown will shutdown the exporter.
func (e *tracesExporter) Shutdown(_ context.Context) error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

func (e *tracesExporter) pushTraceData(ctx context.Context, td ptrace.Traces) error {
	start := time.Now()
	err := doWithTx(ctx, e.client, func(tx *sql.Tx) error {
		statement, err := tx.PrepareContext(ctx, e.insertSQL)
		if err != nil {
			return fmt.Errorf("PrepareContext:%w", err)
		}
		defer func() {
			_ = statement.Close()
		}()
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			spans := td.ResourceSpans().At(i)
			res := spans.Resource()
			resAttr := attributesToMap(res.Attributes())
			var serviceName string
			if v, ok := res.Attributes().Get(conventions.AttributeServiceName); ok {
				serviceName = v.Str()
			}
			for j := 0; j < spans.ScopeSpans().Len(); j++ {
				rs := spans.ScopeSpans().At(j).Spans()
				for k := 0; k < rs.Len(); k++ {
					r := rs.At(k)
					spanAttr := attributesToMap(r.Attributes())
					status := r.Status()
					eventTimes, eventNames, eventAttrs := convertEvents(r.Events())
					linksTraceIDs, linksSpanIDs, linksTraceStates, linksAttrs := convertLinks(r.Links())
					_, err = statement.ExecContext(ctx,
						r.StartTimestamp().AsTime(),
						r.TraceID().HexString(),
						r.SpanID().HexString(),
						r.ParentSpanID().HexString(),
						r.TraceState().AsRaw(),
						r.Name(),
						r.Kind().String(),
						serviceName,
						resAttr,
						spanAttr,
						r.EndTimestamp().AsTime().Sub(r.StartTimestamp().AsTime()).Nanoseconds(),
						status.Code().String(),
						status.Message(),
						eventTimes,
						eventNames,
						eventAttrs,
						linksTraceIDs,
						linksSpanIDs,
						linksTraceStates,
						linksAttrs,
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
	e.logger.Info("insert traces", zap.Int("records", td.SpanCount()),
		zap.String("cost", duration.String()))
	return err
}

func convertEvents(events ptrace.SpanEventSlice) ([]time.Time, []string, []map[string]string) {
	var (
		times []time.Time
		names []string
		attrs []map[string]string
	)
	for i := 0; i < events.Len(); i++ {
		event := events.At(i)
		times = append(times, event.Timestamp().AsTime())
		names = append(names, event.Name())
		attrs = append(attrs, attributesToMap(event.Attributes()))
	}
	return times, names, attrs
}

func convertLinks(links ptrace.SpanLinkSlice) ([]string, []string, []string, []map[string]string) {
	var (
		traceIDs []string
		spanIDs  []string
		states   []string
		attrs    []map[string]string
	)
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		traceIDs = append(traceIDs, link.TraceID().HexString())
		spanIDs = append(spanIDs, link.SpanID().HexString())
		states = append(states, link.TraceState().AsRaw())
		attrs = append(attrs, attributesToMap(link.Attributes()))
	}
	return traceIDs, spanIDs, states, attrs
}

const (
	// language=ClickHouse SQL
	createTracesTableSQL = `
CREATE TABLE IF NOT EXISTS %s (
     Timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
     TraceId String CODEC(ZSTD(1)),
     SpanId String CODEC(ZSTD(1)),
     ParentSpanId String CODEC(ZSTD(1)),
     TraceState String CODEC(ZSTD(1)),
     SpanName LowCardinality(String) CODEC(ZSTD(1)),
     SpanKind LowCardinality(String) CODEC(ZSTD(1)),
     ServiceName LowCardinality(String) CODEC(ZSTD(1)),
     ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     SpanAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
     Duration Int64 CODEC(ZSTD(1)),
     StatusCode LowCardinality(String) CODEC(ZSTD(1)),
     StatusMessage String CODEC(ZSTD(1)),
     Events Nested (
         Timestamp DateTime64(9),
         Name LowCardinality(String),
         Attributes Map(LowCardinality(String), String)
     ) CODEC(ZSTD(1)),
     Links Nested (
         TraceId String,
         SpanId String,
         TraceState String,
         Attributes Map(LowCardinality(String), String)
     ) CODEC(ZSTD(1)),
     INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
     INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_span_attr_key mapKeys(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_span_attr_value mapValues(SpanAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
     INDEX idx_duration Duration TYPE minmax GRANULARITY 1
) ENGINE MergeTree()
%s
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toUnixTimestamp(Timestamp), TraceId)
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	insertTracesSQLTemplate = `INSERT INTO %s (
                        Timestamp,
                        TraceId,
                        SpanId,
                        ParentSpanId,
                        TraceState,
                        SpanName,
                        SpanKind,
                        ServiceName,
                        ResourceAttributes,
                        SpanAttributes,
                        Duration,
                        StatusCode,
                        StatusMessage,
                        Events.Timestamp,
                        Events.Name,
                        Events.Attributes,
                        Links.TraceId,
                        Links.SpanId,
                        Links.TraceState,
                        Links.Attributes
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
                                  ?,
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

const (
	createTraceIDTsTableSQL = `
create table IF NOT EXISTS %s_trace_id_ts (
     TraceId String CODEC(ZSTD(1)),
     Start DateTime CODEC(ZSTD(1)),
     End DateTime CODEC(ZSTD(1)),
     INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE MergeTree()
%s
ORDER BY (TraceId, toUnixTimestamp(Start))
SETTINGS index_granularity=8192;
`
	createTraceIDTsMaterializedViewSQL = `
CREATE MATERIALIZED VIEW IF NOT EXISTS %s_trace_id_ts_mv
TO %s.%s_trace_id_ts
AS SELECT
TraceId,
min(toDateTime(Timestamp)) as Start,
max(toDateTime(Timestamp)) as End
FROM
%s.%s
WHERE TraceId!=''
GROUP BY TraceId;
`
)

func createTracesTable(cfg *Config, db *sql.DB) error {
	if _, err := db.Exec(renderCreateTracesTableSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traces table sql: %w", err)
	}
	if _, err := db.Exec(renderCreateTraceIDTsTableSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traceIDTs table sql: %w", err)
	}
	if _, err := db.Exec(renderTraceIDTsMaterializedViewSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traceIDTs view sql: %w", err)
	}
	return nil
}

func renderInsertTracesSQL(cfg *Config) string {
	return fmt.Sprintf(strings.ReplaceAll(insertTracesSQLTemplate, "'", "`"), cfg.TracesTableName)
}

func renderCreateTracesTableSQL(cfg *Config) string {
	var ttlExpr string
	if cfg.TTLDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(Timestamp) + toIntervalDay(%d)`, cfg.TTLDays)
	}
	return fmt.Sprintf(createTracesTableSQL, cfg.TracesTableName, ttlExpr)
}

func renderCreateTraceIDTsTableSQL(cfg *Config) string {
	var ttlExpr string
	if cfg.TTLDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(Start) + toIntervalDay(%d)`, cfg.TTLDays)
	}
	return fmt.Sprintf(createTraceIDTsTableSQL, cfg.TracesTableName, ttlExpr)
}

func renderTraceIDTsMaterializedViewSQL(cfg *Config) string {
	database, _ := parseDSNDatabase(cfg.DSN)
	return fmt.Sprintf(createTraceIDTsMaterializedViewSQL, cfg.TracesTableName,
		database, cfg.TracesTableName, database, cfg.TracesTableName)
}
