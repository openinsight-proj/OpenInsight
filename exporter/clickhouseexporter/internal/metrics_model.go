package internal

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"strings"
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
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
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
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
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
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
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
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
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
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
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

/*

 */

var supportMetricsType = [...]string{createGaugeTableSQL, createSumTableSQL, createHistogramTableSQL, createExpHistogramTableSQL, createSummaryTableSQL}

type MetricsModel interface {
	Add(metrics interface{}, name string, description string, unit string)
	InjectMetaData(metaData *MetricsMetaData)
	Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error
}

type MetricsMetaData struct {
	ResAttr    map[string]string
	ResUrl     string
	ScopeUrl   string
	ScopeInstr pcommon.InstrumentationScope
}

func CreateMetricsTable(tableName string, TTLDays uint, db *sql.DB) error {
	var ttlExpr string
	if TTLDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(TimeUnix) + toIntervalDay(%d)`, TTLDays)
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
	for _, metrics := range metricsMap {
		if err := metrics.Insert(ctx, tx, logger); err != nil {
			return err
		}
	}
	return nil
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
		attrs = append(attrs, attributesToMap(exemplar.FilteredAttributes()))
		times = append(times, exemplar.Timestamp().AsTime())
		floatValues = append(floatValues, exemplar.DoubleValue())
		intValues = append(intValues, exemplar.IntValue())
		traceIDs = append(traceIDs, exemplar.TraceID().HexString())
		spanIDs = append(spanIDs, exemplar.SpanID().HexString())
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
	switch slice.(type) {
	case []uint64:
		s, _ := slice.([]uint64)
		for _, item := range s {
			set = append(set, item)
		}
	case []float64:
		s, _ := slice.([]float64)
		for _, item := range s {
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
