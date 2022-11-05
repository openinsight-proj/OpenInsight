# metric数据库设计

## MetaData Filed

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

```

## NumberDataPoint Filed

```sql
_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_value_as_double Float64 CODEC(ZSTD(1),
_value_as_int UInt32 CODEC(ZSTD(1)),
_flags UInt32  CODEC(ZSTD(1)),
_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
```

## HistogramDataPoint Filed

```sql
_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_count Int64 CODEC(Delta, ZSTD(1)),
_sum Float64 CODEC(ZSTD(1),
_bucket_counts array(Float64) CODEC(ZSTD(1),
_explicit_bounds array(Float64) CODEC(ZSTD(1),
_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
_flags UInt32  CODEC(ZSTD(1)),
_min Float64 CODEC(ZSTD(1),
_max Float64 CODEC(ZSTD(1),
```

## ExponentialHistogramDataPoint Filed

```sql
_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
_count Int64 CODEC(Delta, ZSTD(1)),
_sum Float64 CODEC(ZSTD(1),
_scale Int32 CODEC(ZSTD(1),
_zero_count Float64 CODEC(ZSTD(1),
_positive_offset Int32 CODEC(ZSTD(1),
_positive_bucket_counts array(UInt32) CODEC(ZSTD(1),
_negative_offset Int32 CODEC(ZSTD(1),
_negative_bucket_counts array(UInt32) CODEC(ZSTD(1),
_flags UInt32  CODEC(ZSTD(1)),
_exemplars Nested (
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

gauge_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
gauge_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
gauge_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
gauge_value_as_double Float64 CODEC(ZSTD(1),
gauge_value_as_int UInt32 CODEC(ZSTD(1)),
gauge_flags UInt32  CODEC(ZSTD(1)),
gauge_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)
```

#### Table Creation SQL

```
TUDO...
```

### Sum

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

sum_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
sum_start_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
sum_time_unix_nano DateTime64(9) CODEC(Delta, ZSTD(1)),
sum_value_as_double Float64 CODEC(ZSTD(1),
sum_value_as_int UInt32 CODEC(ZSTD(1)),
sum_flags UInt32  CODEC(ZSTD(1)),
sum_exemplars Nested (
    filtered_attributes Map(LowCardinality(String), String),
    time_unix_nano DateTime64(9),
    value_as_double Float64,
  	value_as_int UInt32,
    span_id String,
    trace_id String
)

sum_aggregation_temporality String CODEC(ZSTD(1)),
sum_is_monotonic Boolean CODEC(Delta, ZSTD(1)),
```

#### Table Creation SQL

```
TUDO ...
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



