package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
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
    ScopeDroppedAttributesCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    ValueAsDouble Float64 CODEC(ZSTD(1)),
    ValueAsInt Int64 CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1)),
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
    ScopeDroppedAttributesCount UInt32 CODEC(ZSTD(1)),
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
	createHistogramSQL = `
CREATE TABLE IF NOT EXISTS %s_histogram (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttributesCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
	StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
	TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    Count Int64 CODEC(Delta, ZSTD(1)),
    Sum Float64 CODEC(ZSTD(1),
    BucketCounts array(Float64) CODEC(ZSTD(1),
    ExplicitBounds array(Float64) CODEC(ZSTD(1),
	Exemplars Nested (
	FilteredAttributes Map(LowCardinality(String), String),
	TimeUnix DateTime64(9),
	ValueAsDouble Float64,
	ValueAsInt Int64,
	SpanId String,
	TraceId String
    ) CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1)),
    Min Float64 CODEC(ZSTD(1),
    Max Float64 CODEC(ZSTD(1),
    AggTemp Int32 CODEC(ZSTD(1)),
) ENGINE MergeTree()
%s
PARTITION BY toUnixTimestamp64Nano(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
)

const (
	// language=ClickHouse SQL
	insertGaugeTableSQL = `
INSERT INTO %s_gauge (
    ResourceAttributes,
    ResourceSchemaUrl,
    ScopeName,
    ScopeVersion,
    ScopeAttributes,
    ScopeSchemaUrl,
    MetricName,
    MetricDescription,
    MetricUnit,
    Attributes,
    TimeUnix,
    ValueAsDouble,
    ValueAsInt,
    Flags) VALUES `
	// language=ClickHouse SQL
	insertSumTableSQL = `
INSERT INTO %s_sum (
    ResourceAttributes,
    ResourceSchemaUrl,
    ScopeName,
    ScopeVersion,
    ScopeAttributes,
    ScopeSchemaUrl,
    MetricName,
    MetricDescription,
    MetricUnit,
    Attributes,
    TimeUnix,
    ValueAsDouble,
    ValueAsInt,
    Flags,
	AggTemp,
	IsMonotonic) VALUES `
)

/*
   Exemplars.FilteredAttributes,
   Exemplars.TimeUnix,
   Exemplars.ValueAsDouble,
   Exemplars.ValueAsInt,
   Exemplars.SpanId,
   Exemplars.TraceId
*/

var supportMetricsType = [...]string{createGaugeTableSQL, createSumTableSQL}

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
	//todo others
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

func convertExemplars(exemplars pmetric.ExemplarSlice) ([]map[string]string, []time.Time, []float64, []int64, []string, []string) {
	var (
		attrs       []map[string]string
		times       []time.Time
		floatValues []float64
		intValues   []int64
		traceIDs    []string
		spanIDs     []string
	)
	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		for j := 0; j < exemplar.FilteredAttributes().Len(); j++ {
			attrs = append(attrs, attributesToMap(exemplar.FilteredAttributes()))
		}
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
