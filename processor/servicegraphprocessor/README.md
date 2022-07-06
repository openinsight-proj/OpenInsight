# Service graph processor

| Status                   |                |
| ------------------------ |----------------|
| Stability                | [alpha]         |
| Supported pipeline types | traces         |
| Distributions            | [contrib] |

The service graphs processor is a traces processor that builds a map representing the interrelationships between various services in a system. 
The processor will analyse trace data and generate metrics describing the relationship between the services. 
These metrics can be used by data visualization apps to draw a service graph.

# How it works
This processor works by inspecting spans and looking for the tag [span.kind](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind). If it finds the span kind to be CLIENT or SERVER, it stores the request in a local in-memory store.
That request waits until its corresponding client or server pair span is processed or until the maximum waiting time has passed. When either of those conditions is reached, the request is processed and removed from the local store. If the request is complete by that time, it’ll be recorded as an edge in the graph.
Edges are represented as metrics, while nodes in the graphs are recorded as client and server labels in the metric.

e.g. if service A (client) makes a request to service B (server), that metric will get recorded as a timeseries in metric traces_service_graph_request_total. In Prometheus representation:
```bash
traces_service_graph_request_total{client="A",server="B"} 1
```

Since the service graph processor has to process both sides of an edge, it needs to process all spans of a trace to function properly. If spans of a trace are spread out over multiple pipelines it will not be possible to pair up spans reliably.

# Example configuration for the component
```yaml
processors:
  service_graph:
    metrics_exporter: metrics
    latency_histogram_buckets: [1,2,3,4,5]
    dimensions:
      - dimension-1
      - dimension-2
    store:
      ttl: 1s
      max_items: 10
```

### More Examples

For more example configuration covering various other use cases, please visit the [testdata directory](./testdata).

[alpha]: https://github.com/open-telemetry/opentelemetry-collector#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
