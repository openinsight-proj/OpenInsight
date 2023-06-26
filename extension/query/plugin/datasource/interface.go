package datasource

import (
	"context"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/v1alpha1"
	semconv "go.opentelemetry.io/collector/semconv/v1.6.1"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_resource "go.opentelemetry.io/proto/otlp/resource/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"time"
)

type Query interface {
	GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error)
	SearchTraces(ctx context.Context, query *TraceQueryParameters) (*v1alpha1.TracesData, error)
	SearchLogs(ctx context.Context) (*v1_logs.LogsData, error)
	GetLog(ctx context.Context) (*v1_logs.LogsData, error)
	GetService(ctx context.Context) ([]*v1_resource.Resource, error)
	GetOperations(ctx context.Context, query *OperationsQueryParameters) ([]string, error)

	//TODO: add metrics query.
}

// TraceQueryParameters contains parameters of a trace query.
type TraceQueryParameters struct {
	ServiceName   string
	OperationName string
	Tags          map[string]string
	StartTime     time.Time
	EndTime       time.Time
	DurationMin   *duration.Duration
	DurationMax   *duration.Duration
	NumTraces     int
}

type OperationsQueryParameters struct {
	ServiceName string
	// optional
	SpanKind string
}

func FindRootSpan(spans []*v1_trace.Span) *v1_trace.Span {
	for _, span := range spans {
		parentSpanId := string(span.ParentSpanId)

		hasParent := false
		for _, subSpan := range spans {
			if parentSpanId == string(subSpan.SpanId) {
				hasParent = true
				break
			}
		}

		if !hasParent {
			return span
		}
	}
	return nil
}

// DocumentsTracesConvert will convert Otel tracesData into openinsight trace list data.
func DocumentsTracesConvert(otlpTraces *v1_trace.TracesData) (*v1alpha1.TracesData, error) {
	var traces []*v1alpha1.Trace

	// traceId ===> []ResourceSpans
	//spansMap := make(map[string][]*v1_trace.Span)
	spansMap := orderedmap.NewOrderedMap[string, []*v1_trace.Span]()
	// traceId ===> []Resource)
	resourceMap := make(map[string][]*v1_resource.Resource)

	// group resourceSpans by traceId
	for _, rSpan := range otlpTraces.ResourceSpans {
		traceId := string(rSpan.ScopeSpans[0].Spans[0].TraceId)
		if oldSpans, found := spansMap.Get(traceId); found {
			spansMap.Set(traceId, append(oldSpans, rSpan.ScopeSpans[0].Spans...))
		} else {
			spansMap.Set(traceId, rSpan.ScopeSpans[0].Spans)
		}

		for _, keyValue := range rSpan.Resource.Attributes {
			if keyValue.Key == semconv.AttributeServiceName {
				if oldRes, found := resourceMap[traceId]; found {
					for _, re := range oldRes {
						for _, attribute := range re.Attributes {
							// if resource with service's name exists. ignore it, otherwise, append
							if semconv.AttributeServiceName == attribute.Key {
								if attribute.Value.GetStringValue() != keyValue.Value.GetStringValue() {
									resourceMap[traceId] = append(oldRes, rSpan.Resource)
								}
							}
						}
					}
				} else {
					resourceMap[traceId] = []*v1_resource.Resource{rSpan.Resource}
				}
			}
		}
	}

	for _, id := range spansMap.Keys() {
		resources := resourceMap[id]
		var process []*v1alpha1.Trace_ResourceProcess
		for _, resource := range resources {
			pro := new(v1alpha1.Process)
			for _, attribute := range resource.Attributes {
				if attribute.Key == "service.name" {
					pro.ServiceName = attribute.Value.GetStringValue()
				} else {
					pro.Tags = append(pro.Tags, &v1alpha1.KeyValue{
						Key:  attribute.Key,
						VStr: attribute.Value.GetStringValue(),
					})
				}
			}
			process = append(process, &v1alpha1.Trace_ResourceProcess{Process: pro})
		}
		spans, _ := spansMap.Get(id)

		trace := &v1alpha1.Trace{
			TraceId:    id,
			SpanCount:  uint32(len(spans)),
			ProcessMap: process,
		}

		rootSpan := FindRootSpan(spans)
		if rootSpan != nil {
			trace.OperationName = rootSpan.Name
			trace.Warnings = []string{}
			// TODO(jian): why span statue nil?
			if rootSpan.Status != nil {
				switch rootSpan.Status.Code {
				case v1_trace.Status_STATUS_CODE_OK:
					trace.Status = v1alpha1.Trace_HEALTHY
				case v1_trace.Status_STATUS_CODE_ERROR:
					trace.Status = v1alpha1.Trace_UNHEALTHY
				case v1_trace.Status_STATUS_CODE_UNSET:
					if rootSpan.Kind == v1_trace.Span_SPAN_KIND_CLIENT {
						trace.Status = v1alpha1.Trace_UNHEALTHY
					}

					if rootSpan.Kind == v1_trace.Span_SPAN_KIND_SERVER || rootSpan.Kind == v1_trace.Span_SPAN_KIND_INTERNAL {
						trace.Status = v1alpha1.Trace_HEALTHY
					}
					// TODO(jian): others
				default:
					trace.Status = v1alpha1.Trace_UNHEALTHY
				}
			} else {
				trace.Status = v1alpha1.Trace_UNHEALTHY
			}
			trace.StartTime = time.UnixMicro(int64(rootSpan.StartTimeUnixNano / 1000)).String()
			trace.Duration = time.Duration(rootSpan.EndTimeUnixNano - rootSpan.StartTimeUnixNano).String()
		}

		traces = append(traces, trace)
	}

	return &v1alpha1.TracesData{Traces: traces}, nil
}
