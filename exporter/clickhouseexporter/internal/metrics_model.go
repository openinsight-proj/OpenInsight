// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter/internal"

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	// language=ClickHouse SQL
	createGaugeTableSQL = `
CREATE TABLE IF NOT EXISTS %s_gauge (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC (ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    ValueAsDouble Float64 CODEC(ZSTD(1)),
    ValueAsInt Int64 CODEC(ZSTD(1)),
    Flags UInt32 CODEC(ZSTD(1)),
    Exemplars Nested (
		FilteredAttributes Map(LowCardinality(String), String),
		TimeUnix DateTime64(9),
		ValueAsDouble Float64,
		ValueAsInt Int64,
		SpanId String,
		TraceId String
    ) CODEC(ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	createSumTableSQL = `
CREATE TABLE IF NOT EXISTS %s_sum (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
	ServiceName LowCardinality(String) CODEC (ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
	StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	ValueAsDouble Float64 CODEC(ZSTD(1)),
	ValueAsInt UInt32 CODEC(ZSTD(1)),
	Flags UInt32  CODEC(ZSTD(1)),
    Exemplars Nested (
		FilteredAttributes Map(LowCardinality(String), String),
		TimeUnix DateTime64(9),
		ValueAsDouble Float64,
		ValueAsInt Int64,
		SpanId String,
		TraceId String
    ) CODEC(ZSTD(1)),
    AggTemp Int32 CODEC(ZSTD(1)),
	IsMonotonic Boolean CODEC(Delta, ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	createHistogramTableSQL = `
CREATE TABLE IF NOT EXISTS %s_histogram (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC (ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
	StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    Count Int64 CODEC(Delta, ZSTD(1)),
    Sum Float64 CODEC(ZSTD(1)),
    BucketCounts Array(UInt64) CODEC(ZSTD(1)),
    ExplicitBounds Array(Float64) CODEC(ZSTD(1)),
	Exemplars Nested (
		FilteredAttributes Map(LowCardinality(String), String),
		TimeUnix DateTime64(9),
		ValueAsDouble Float64,
		ValueAsInt Int64,
		SpanId String,
		TraceId String
    ) CODEC(ZSTD(1)),
    Flags UInt32 CODEC(ZSTD(1)),
    Min Float64 CODEC(ZSTD(1)),
    Max Float64 CODEC(ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	createExpHistogramTableSQL = `
CREATE TABLE IF NOT EXISTS %s_exponential_histogram (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC (ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
	StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    Count Int64 CODEC(Delta, ZSTD(1)),
    Sum Float64 CODEC(ZSTD(1)),
    Scale Int32 CODEC(ZSTD(1)),
    ZeroCount UInt64 CODEC(ZSTD(1)),
	PositiveOffset Int32 CODEC(ZSTD(1)),
	PositiveBucketCounts Array(UInt64) CODEC(ZSTD(1)),
	NegativeOffset Int32 CODEC(ZSTD(1)),
	NegativeBucketCounts Array(UInt64) CODEC(ZSTD(1)),
	Exemplars Nested (
		FilteredAttributes Map(LowCardinality(String), String),
		TimeUnix DateTime64(9),
		ValueAsDouble Float64,
		ValueAsInt Int64,
		SpanId String,
		TraceId String
    ) CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1)),
    Min Float64 CODEC(ZSTD(1)),
    Max Float64 CODEC(ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	createSummaryTableSQL = `
CREATE TABLE IF NOT EXISTS %s_summary (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttrCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    ServiceName LowCardinality(String) CODEC (ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
	StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    Count UInt64 CODEC(Delta, ZSTD(1)),
    Sum Float64 CODEC(ZSTD(1)),
    ValueAtQuantiles Nested(
		Quantile Float64,
		Value Float64
	) CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (ServiceName, MetricName, Attributes, toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
)

const (
	// language=ClickHouse SQL
	insertGaugeTableSQL = `INSERT INTO %s_gauge (
	ResourceAttributes,
	ResourceSchemaUrl,
	ScopeName,
	ScopeVersion,
	ScopeAttributes,
	ScopeDroppedAttrCount,
	ScopeSchemaUrl,
	ServiceName,
	MetricName,
	MetricDescription,
	MetricUnit,
	Attributes,
	TimeUnix,
	ValueAsDouble,
	ValueAsInt,
	Flags,
	Exemplars.FilteredAttributes,
	Exemplars.TimeUnix,
	Exemplars.ValueAsDouble,
	Exemplars.ValueAsInt,
	Exemplars.SpanId,
	Exemplars.TraceId) VALUES `
	// language=ClickHouse SQL
	insertSumTableSQL = `INSERT INTO %s_sum (
	ResourceAttributes,
	ResourceSchemaUrl,
	ScopeName,
	ScopeVersion,
	ScopeAttributes,
	ScopeDroppedAttrCount,
	ScopeSchemaUrl,
	ServiceName,
	MetricName,
	MetricDescription,
	MetricUnit,
	Attributes,
	TimeUnix,
	ValueAsDouble,
	ValueAsInt,
	Flags,
	Exemplars.FilteredAttributes,
	Exemplars.TimeUnix,
	Exemplars.ValueAsDouble,
	Exemplars.ValueAsInt,
	Exemplars.SpanId,
	Exemplars.TraceId,
	AggTemp,
	IsMonotonic) VALUES `
	// language=ClickHouse SQL
	insertHistogramTableSQL = `INSERT INTO %s_histogram (
	ResourceAttributes,
	ResourceSchemaUrl,
	ScopeName,
	ScopeVersion,
	ScopeAttributes,
	ScopeDroppedAttrCount,
	ScopeSchemaUrl,
	ServiceName,
	MetricName,
	MetricDescription,
	MetricUnit,
	Attributes,
	StartTimeUnix,
	TimeUnix,
	Count,
	Sum,
	BucketCounts,
	ExplicitBounds,
	Exemplars.FilteredAttributes,
	Exemplars.TimeUnix,
	Exemplars.ValueAsDouble,
	Exemplars.ValueAsInt,
	Exemplars.SpanId,
	Exemplars.TraceId,
	Flags,
	Min,
	Max) VALUES `
	// language=ClickHouse SQL
	insertExpHistogramTableSQL = `INSERT INTO %s_exponential_histogram (
	ResourceAttributes,
	ResourceSchemaUrl,
	ScopeName,
	ScopeVersion,
	ScopeAttributes,
	ScopeDroppedAttrCount,
	ScopeSchemaUrl,
	ServiceName,
	MetricName,
	MetricDescription,
	MetricUnit,
	Attributes,
	StartTimeUnix,
	TimeUnix,
	Count,
	Sum,                   
	Scale,
	ZeroCount,
	PositiveOffset,
	PositiveBucketCounts,
	NegativeOffset,
	NegativeBucketCounts,
	Exemplars.FilteredAttributes,
	Exemplars.TimeUnix,
	Exemplars.ValueAsDouble,
	Exemplars.ValueAsInt,
	Exemplars.SpanId,
	Exemplars.TraceId,
	Flags,
	Min,
	Max) VALUES `
	// language=ClickHouse SQL
	insertSummaryTableSQL = `INSERT INTO %s_summary (
	ResourceAttributes,
	ResourceSchemaUrl,
	ScopeName,
	ScopeVersion,
	ScopeAttributes,
	ScopeDroppedAttrCount,
	ScopeSchemaUrl,
	ServiceName,
	MetricName,
	MetricDescription,
	MetricUnit,
	Attributes,
	StartTimeUnix,
	TimeUnix,
	Count,
	Sum,
	ValueAtQuantiles.Quantile,
	ValueAtQuantiles.Value,
	Flags) VALUES `
)

var supportMetricsType = [...]string{createGaugeTableSQL, createSumTableSQL, createHistogramTableSQL, createExpHistogramTableSQL, createSummaryTableSQL}

type MetricsModel interface {
	Add(metrics interface{}, name string, description string, unit string)
	InjectMetaData(metaData *MetricsMetaData)
	Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error
}

type MetricsMetaData struct {
	ServiceName string
	ResAttr     map[string]string
	ResURL      string
	ScopeURL    string
	ScopeInstr  pcommon.InstrumentationScope
}

func CreateMetricsTable(tableName string, ttlDays uint, db *sql.DB) error {
	var ttlExpr string
	if ttlDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(TimeUnix) + toIntervalDay(%d)`, ttlDays)
	}
	for _, table := range supportMetricsType {
		query := fmt.Sprintf(table, tableName, ttlExpr)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("exec create gauge table sql: %w", err)
		}
	}
	return nil
}

func CreateMetricsModel(tableName string) map[pmetric.MetricType]MetricsModel {
	metricsMap := make(map[pmetric.MetricType]MetricsModel)
	metricsMap[pmetric.MetricTypeGauge] = &GaugeMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertGaugeTableSQL, "'", "`"), tableName),
	}
	metricsMap[pmetric.MetricTypeSum] = &SumMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertSumTableSQL, "'", "`"), tableName),
	}
	metricsMap[pmetric.MetricTypeHistogram] = &HistogramMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertHistogramTableSQL, "'", "`"), tableName),
	}
	metricsMap[pmetric.MetricTypeExponentialHistogram] = &ExpHistogramMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertExpHistogramTableSQL, "'", "`"), tableName),
	}
	metricsMap[pmetric.MetricTypeSummary] = &SummaryMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertSummaryTableSQL, "'", "`"), tableName),
	}
	return metricsMap
}

func InjectMetaData(metricsMap map[pmetric.MetricType]MetricsModel, metaData *MetricsMetaData) {
	for _, metrics := range metricsMap {
		metrics.InjectMetaData(metaData)
	}
}

func InsertMetrics(ctx context.Context, tx *sql.Tx, metricsMap map[pmetric.MetricType]MetricsModel, logger *zap.Logger) error {
	var errs []error
	for _, metrics := range metricsMap {
		errs = append(errs, metrics.Insert(ctx, tx, logger))
	}
	return multierr.Combine(errs...)
}

func convertExemplars(exemplars pmetric.ExemplarSlice) (clickhouse.ArraySet, clickhouse.ArraySet, clickhouse.ArraySet, clickhouse.ArraySet, clickhouse.ArraySet, clickhouse.ArraySet) {
	var (
		attrs       clickhouse.ArraySet
		times       clickhouse.ArraySet
		floatValues clickhouse.ArraySet
		intValues   clickhouse.ArraySet
		traceIDs    clickhouse.ArraySet
		spanIDs     clickhouse.ArraySet
	)
	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		if !exemplar.TraceID().IsEmpty() && !exemplar.SpanID().IsEmpty() {
			attrs = append(attrs, attributesToMap(exemplar.FilteredAttributes()))
			times = append(times, exemplar.Timestamp().AsTime())
			floatValues = append(floatValues, exemplar.DoubleValue())
			intValues = append(intValues, exemplar.IntValue())
			traceID := exemplar.TraceID()
			traceIDs = append(traceIDs, hex.EncodeToString(traceID[:]))
			spanID := exemplar.SpanID()
			spanIDs = append(spanIDs, hex.EncodeToString(spanID[:]))
		}
	}
	return attrs, times, floatValues, intValues, traceIDs, spanIDs
}

func attributesToMap(attributes pcommon.Map) map[string]string {
	m := make(map[string]string, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		m[k] = v.Str()
		return true
	})
	return m
}

func convertSliceToArraySet(slice interface{}) clickhouse.ArraySet {
	var set clickhouse.ArraySet
	switch slice := slice.(type) {
	case []uint64:
		for _, item := range slice {
			set = append(set, item)
		}
	case []float64:
		for _, item := range slice {
			set = append(set, item)
		}
	}
	return set
}

func convertValueAtQuantile(valueAtQuantile pmetric.SummaryDataPointValueAtQuantileSlice) (clickhouse.ArraySet, clickhouse.ArraySet) {
	var (
		quantiles clickhouse.ArraySet
		values    clickhouse.ArraySet
	)
	for i := 0; i < valueAtQuantile.Len(); i++ {
		value := valueAtQuantile.At(i)
		quantiles = append(quantiles, value.Quantile())
		values = append(values, value.Value())
	}
	return quantiles, values
}
