# Clickhouse storage

## SQL design
```sql
--1. 先进行 Duration, Start 查询所有符合traceid,并安时间排序，最后使用limit:  
--(SUBSQL)
SELECT TraceId AS id FROM otel.otel_traces_trace_id_ts_mv WHERE Start BETWEEN '2022-10-23 23:56:18' AND '2022-10-23 23:56:21' AND (End - Start) BETWEEN 20000000 AND 100000000 ORDER BY Start DESC LIMIT 20
                                                                                                                                                                               
--2.连接 SUBSQL 和 otel_traces并添加其他查询条件并查询数据
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