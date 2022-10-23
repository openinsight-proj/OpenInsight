# Clickhouse storage

## SQL design
```sql
--1. 先进行时间赛选器 ，找到筛选出来的所有符合traceid:  
SELECT TraceId,Start,(End - Start) AS Duration FROM otel.otel_traces_trace_id_ts_mv WHERE '2022-10-23 04:35:16'<= Start AND Start <= '2022-10-23 04:35:22' And 0<= Duration AND Duration <=0 ORDER BY Start DESC


--2. 再匹配各种查询字段 得到最终需要的 traceId
SELECT TraceId FROM otel.otel_traces WHERE ServiceName='this service [1]'

--2.1 组合时间条件与其他条件,Limit 分页
SELECT a.TraceId FROM 
(SELECT TraceId,Start,(End - Start) Duration FROM otel.otel_traces_trace_id_ts_mv WHERE '2022-10-23 04:35:16'<= Start AND Start <= '2022-10-23 04:35:22' And 0<= Duration AND Duration <=0 ORDER BY Start DESC) AS a JOIN
(SELECT TraceId FROM otel.otel_traces WHERE ServiceName='this service [1]') AS b ON a.TraceId = b.TraceId LIMIT 20


--3. 查询具体数据，
--3.1 查询具体数据
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
       a.Links.Attributes FROM otel.otel_traces AS a

--3.2 组合已有的查询条件与具体数据，查询出所有的span
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
       (SELECT a.TraceId FROM 
(SELECT TraceId,Start,(End - Start) AS Duration FROM otel.otel_traces_trace_id_ts_mv WHERE '2022-10-23 04:35:16'<= Start AND Start <= '2022-10-23 04:35:22' And 0<= Duration AND Duration <=0 ORDER BY Start DESC) AS a JOIN
(SELECT TraceId FROM otel.otel_traces WHERE ServiceName='this service [1]') AS b ON a.TraceId = b.TraceId LIMIT 20) AS b,
       otel.otel_traces AS a
       WHERE b.TraceId = a.TraceId
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
		这个不能代表一条链路的开始与结束，因为End= 在一条链路中，所有span的最晚开始时间+这个span的duration
	*/
```