# OpenInsight

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]

<br />
<p align="center">
  <a href="https://github.com/openinsight-proj/openinsight">
    <img src="" alt="Logo" width="80" height="80">
  </a>

<h3 align="center">OpenInsight</h3>
  <p align="center">
    You know, OpenTelemetry Collector enhancement distribution
    <br />
    An observability analytics platform for unified and composable data storage. 
    <br />
    <a href=""><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="">Visit our blog</a>
    ·
    <a href="https://github.com/openinsight-proj/openinsight/issues">Report Bug</a>
    ·
    <a href="https://github.com/openinsight-proj/openinsight/issues">Request Feature</a>
  </p>
</p>

With OpenInsight, you can choose different storage databases for different data (Metrics, Tracing, Logging), or you can choose a unified storage database for storage and analysis.

**Our aim is to extend and not to replace the OpenTelemetry Collector.**

# Scenarios
## 1. Clickhouse
You only choose Clickhouse for Logging & Tracing & Metrics store.

With OpenInsight configuration:
```yaml
extensions:
  health_check:
  query:
    protocols:
      http:
        endpoint: 0.0.0.0:8080
    storage:
      clickhouse:
        dsn: tcp://127.0.0.1:9000/openinsight
        tls_setting:
          insecure: true
        tracing_table_name: openinsight_traces
        logging_table_name: openinsight_logs
        metrics_table_name: openinsight_metrics
    tracing_query:
      storage_type: clickhouse
    logging_query:
      storage_type: clickhouse
    metrics_query:
      storage_type: clickhouse
```


## 2. Elasticsearch
You only choose Elasticsearch for Logging & Tracing & Metrics store.

With OpenInsight configuration:
```yaml
extensions:
  health_check:
  pprof:
    endpoint: 0.0.0.0:1777
  zpages:
    endpoint: 0.0.0.0:55679
  query:
    protocols:
      http:
        endpoint: 0.0.0.0:18888
      grpc:
        endpoint: 0.0.0.0:18889
    storage:
      elasticsearch:
        endpoints: [ "https://localhost:9200" ]
        traces_index: otlp_spans
        logs_index: otlp_logs
        #metrics_index: otlp_metrics
        user: elastic
        password: dangerous
    tracing_query:
      storage_type: elasticsearch
    logging_query:
      storage_type: elasticsearch
    metrics_query:
      storage_type: elasticsearch
```

## 3. Elasticsearch and Prometheus/VictoriaMetrics
You may choose Elasticsearch for Logging & Tracing store and Prometheus for Metrics.

With OpenInsight configuration:
```yaml
extensions:
  health_check:
  pprof:
    endpoint: 0.0.0.0:1777
  zpages:
    endpoint: 0.0.0.0:55679
  query:
    protocols:
      http:
        endpoint: 0.0.0.0:18888
      grpc:
        endpoint: 0.0.0.0:18889
    storage:
      elasticsearch:
        endpoints: [ "https://localhost:9200" ]
        traces_index: otlp_spans
        user: elastic
        password: dangerous
      prometheus:
        endpoint: "http://localhost:9090"
    tracing_query:
      storage_type: elasticsearch
    logging_query:
      storage_type: elasticsearch
    metrics_query:
      storage_type: prometheus
```

## 4. Clickhouse,Elasticsearch and Prometheus/VictoriaMetrics
You may choose Clickhouse for Logging, Elasticsearch for Tracing and Prometheus for Metrics.

With OpenInsight configuration:
```yaml
extensions:
  health_check:
  pprof:
    endpoint: 0.0.0.0:1777
  zpages:
    endpoint: 0.0.0.0:55679
  query:
    protocols:
      http:
        endpoint: 0.0.0.0:18888
      grpc:
        endpoint: 0.0.0.0:18889
    storage:
      elasticsearch:
        endpoints: [ "https://localhost:9200" ]
        traces_index: otlp_spans
        user: elastic
        password: dangerous
      clickhouse:
        dsn: tcp://127.0.0.1:9000/openinsight
      prometheus:
        endpoint: "http://localhost:9090"
    tracing_query:
      storage_type: elasticsearch
    logging_query:
      storage_type: clickhouse
    metrics_query:
      storage_type: prometheus
```

## Compatibility
| OpenInsight Version | OTEL COl Contrib Version |
|---------------------|--------------------------|
| v0.0.3              | v0.71.0                  |

This table only show the latest version compatibility, more version compatibility please refer [version compatiblity list](docs/version_compatibility_2023.md)


## Contributors
<a href="https://github.com/openinsight-proj/openinsight/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=openinsight-proj/openinsight" />
</a>

[contributors-shield]: https://img.shields.io/github/contributors/openinsight-proj/openinsight.svg?style=for-the-badge
[contributors-url]: https://github.com/openinsight-proj/openinsight/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/openinsight-proj/openinsight.svg?style=for-the-badge
[forks-url]: https://github.com/openinsight-proj/openinsight/network/members
[stars-shield]: https://img.shields.io/github/stars/openinsight-proj/openinsight.svg?style=for-the-badge
[stars-url]: https://github.com/openinsight-proj/openinsight/stargazers
[issues-shield]: https://img.shields.io/github/issues/openinsight-proj/openinsight.svg?style=for-the-badge
[issues-url]: https://github.com/openinsight-proj/openinsight/issues
