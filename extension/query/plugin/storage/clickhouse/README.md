# Clickhouse storage

## Configuration options

```yaml
clickhouse:
  dsn: tcp://127.0.0.1:9000/default
  tls_setting:
    insecure: false
    cert_file: "path"
    key_file: "path"
```
- `dsn`(no default): The ClickHouse server DSN (Data Source Name), for
  example `tcp://127.0.0.1:9000/default`
  For tcp protocol reference: [ClickHouse/clickhouse-go#dsn](https://github.com/ClickHouse/clickhouse-go#dsn).
  For http protocol
  reference: [ClickHouse/clickhouse-go#http-support-experimental](https://github.com/ClickHouse/clickhouse-go/tree/main#http-support-experimental)
  .
- tls_setting
  - `insecure` (default = `false`): whether to enable client transport security for
    the exporter's connection.
  
    As a result, the following parameters are also required under `tls_setting`:

  - `cert_file` (no default): path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to false.
  - `key_file` (no default): path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to false.
 
  The following settings are optional:

  - `server_name_override` (default = `<missing service name>`): requested by client for virtual hosting.

  more tls Configuration [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)


## SQL design
```sql
--1. filger Duration, Start with limit:  
SELECT TraceId AS id FROM otel.otel_traces_trace_id_ts_mv WHERE Start BETWEEN '2022-10-23 23:56:18' AND '2022-10-23 23:56:21' AND (End - Start) BETWEEN 20000000 AND 100000000 ORDER BY Start DESC LIMIT 20
                                                                                                                                                                               
--2.join SUBSQL and otel_traces
SELECT a.Timestamp,
       a.TraceId,
       a.SpanId,
       a.ParentSpanId,
       a.SpanName,
       a.SpanKind,
       a.ServiceName,
       a.Duration,
       a.StatusCode,
       a.StatusMessage,
       a.SpanAttributes,
       a.ResourceAttributes,
       a.Events.Timestamp,
       a.Events.Name,
       a.Events.Attributes,
       a.Links.TraceId,
       a.Links.SpanId,
       a.Links.TraceState,
       a.Links.Attributes FROM
    (SELECT TraceId AS id FROM otel.otel_traces_trace_id_ts_mv WHERE Start BETWEEN '2022-10-23 23:56:18' AND '2022-10-23 23:56:21' AND (End - Start) BETWEEN 20000000 AND 100000000 ORDER BY Start DESC LIMIT 20) AS b JOIN
    otel.otel_traces AS a on b.id = a.TraceId WHERE a.ServiceName='this service [9]' AND a.SpanName='HTTP PUT' AND a.SpanAttributes['Tag_a']='tag_a_value' AND a.SpanAttributes['Tag_b']='tag_b_value'

```

## 已知问题
otel clickhouse exporter 的表创建的问题：https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/clickhouseexporter/README.md#traces-1
```sql
	/*
		CREATE MATERIALIZED VIEW otel.otel_traces_trace_id_ts_mv TO otel.otel_traces_trace_id_ts
		AS
		SELECT TraceId,
		       min(toDateTime(Timestamp)) AS Start,
		       max(toDateTime(Timestamp)) AS End
		FROM otel.otel_traces
		WHERE TraceId != ''
		GROUP BY TraceId;

		这张表的查询表达中：
		Start=在一个链路中，所有span的最早开始时间
		End=在一条链路中，所有span的最晚开始时间
		这个不能代表一条链路的开始与结束，因为End=在一条链路中，所有span的最晚开始时间+这个span的duration
	*/
```