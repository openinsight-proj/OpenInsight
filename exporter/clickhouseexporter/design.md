# metric数据库设计

## MetaData Filed

```sql
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
```

## NumberDataPoint Filed

```sql
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    ValueAsDouble Float64 CODEC(ZSTD(1)),
    ValueAsInt Int64 CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1)),
    Exemplars Nested (
    filteredAttributes Map(LowCardinality(String), String),
    timeUnix DateTime64(9),
    valueAsDouble Float64,
    valueAsInt Int64,
    spanId String,
    traceId String
    ) CODEC(ZSTD(1))
```

## HistogramDataPoint Filed

```sql
Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
StartTime DateTime64(9) CODEC(Delta, ZSTD(1)),
TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
Count Int64 CODEC(ZSTD(1)),
Sum Float64 CODEC(ZSTD(1),
BucketCounts array(Float64) CODEC(ZSTD(1)),
ExplicitBounds array(Float64) CODEC(ZSTD(1)),
Exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
Flags UInt32  CODEC(ZSTD(1)),
Min Float64 CODEC(ZSTD(1),
Max Float64 CODEC(ZSTD(1),
```

## ExponentialHistogramDataPoint Filed

```sql
Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
Count Int64 CODEC(Delta, ZSTD(1)),
Sum Float64 CODEC(ZSTD(1),
Scale Int32 CODEC(ZSTD(1),
ZeroCount Float64 CODEC(ZSTD(1),
Positive_offset Int32 CODEC(ZSTD(1),
Positive_bucket_counts array(UInt32) CODEC(ZSTD(1),
Negative_offset Int32 CODEC(ZSTD(1),
Negative_bucket_counts array(UInt32) CODEC(ZSTD(1),
Flags UInt32  CODEC(ZSTD(1)),
Exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
_min Float64 CODEC(ZSTD(1),
_max Float64 CODEC(ZSTD(1),
```

## SummaryDataPoint Filed

```sql
_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_count Int64 CODEC(Delta, ZSTD(1)),
_sum Float64 CODEC(ZSTD(1),
_value_at_quantiles Nested(
	quantile Float64,
	value Float64
) 
_flags UInt32  CODEC(ZSTD(1)),
```

## Tables 

### Gauge

```SQL
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
    filteredAttributes Map(LowCardinality(String), String),
    timeUnix DateTime64(9),
    valueAsDouble Float64,
    valueAsInt Int64,
    spanId String,
    traceId String
    ) CODEC(ZSTD(1))
    ) ENGINE MergeTree()
    %s
    PARTITION BY toDate(TimeUnix)
    ORDER BY (toUnixTimestamp64Nano(TimeUnix))
    SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
```

### Sum
```sql
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
PARTITION BY toDate(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
```

### Histogram

#### Filed

```sql
ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Resource_schema_url String CODEC(ZSTD(1)),

Scope_Name String CODEC(ZSTD(1)),
Scope_Version String CODEC(ZSTD(1)),
Scope_Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Scope_Dropped_Attributes_Count UInt32 CODEC(ZSTD(1)),
Scope_schema_url String CODEC(ZSTD(1)),

metric_name String CODEC(ZSTD(1)),
metric_description String CODEC(ZSTD(1)),
metric_unit String CODEC(ZSTD(1)),

hisg_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
hisg_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
hisg_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
hisg_count Int64 CODEC(Delta, ZSTD(1)),
hisg_sum Float64 CODEC(ZSTD(1),
hisg_bucket_counts array(Float64) CODEC(ZSTD(1),
hisg_explicit_bounds array(Float64) CODEC(ZSTD(1),
hisg_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
hisg_flags UInt32  CODEC(ZSTD(1)),
hisg_min Float64 CODEC(ZSTD(1),
hisg_max Float64 CODEC(ZSTD(1),

hisg_agg_temp String CODEC(ZSTD(1)),
```

#### Table Creation SQL

```

```

### ExponentialHistogram

#### Filed

```sql
ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Resource_schema_url String CODEC(ZSTD(1)),

Scope_Name String CODEC(ZSTD(1)),
Scope_Version String CODEC(ZSTD(1)),
Scope_Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Scope_Dropped_Attributes_Count UInt32 CODEC(ZSTD(1)),
Scope_schema_url String CODEC(ZSTD(1)),

metric_name String CODEC(ZSTD(1)),
metric_description String CODEC(ZSTD(1)),
metric_unit String CODEC(ZSTD(1)),

eh_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
eh_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
eh_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
eh_count Int64 CODEC(Delta, ZSTD(1)),
eh_sum Float64 CODEC(ZSTD(1),
eh_scale Int32 CODEC(ZSTD(1),
eh_zero_count Float64 CODEC(ZSTD(1),
eh_positive_offset Int32 CODEC(ZSTD(1),
eh_positive_bucket_counts array(UInt32) CODEC(ZSTD(1),
eh_negative_offset Int32 CODEC(ZSTD(1),
eh_negative_bucket_counts array(UInt32) CODEC(ZSTD(1),
eh_flags UInt32  CODEC(ZSTD(1)),
eh_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
eh_aggregation_min Float64 CODEC(ZSTD(1),
eh_max Float64 CODEC(ZSTD(1),

eh_agg_temp String CODEC(ZSTD(1)),
```

#### Table Create SQL

```
TUDO...
```

### Summary

#### Filed

```sql
ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Resource_schema_url String CODEC(ZSTD(1)),

Scope_Name String CODEC(ZSTD(1)),
Scope_Version String CODEC(ZSTD(1)),
Scope_Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
Scope_Dropped_Attributes_Count UInt32 CODEC(ZSTD(1)),
Scope_schema_url String CODEC(ZSTD(1)),

metric_name String CODEC(ZSTD(1)),
metric_description String CODEC(ZSTD(1)),
metric_unit String CODEC(ZSTD(1)),

smary_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
smary_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
smary_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
smary_count Int64 CODEC(Delta, ZSTD(1)),
smary_sum Float64 CODEC(ZSTD(1),
smary_value_at_quantiles Nested(
	quantile Float64,
	value Float64
) 
smary_flags UInt32  CODEC(ZSTD(1)),
```

#### Table Creation SQL

```
TUDO...
```



