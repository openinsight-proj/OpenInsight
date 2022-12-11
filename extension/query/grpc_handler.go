package query

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
)

type Handler struct {
	QueryService *QueryService
}

func (t *Handler) GetOperations(ctx context.Context, req *v1alpha1.GetOperationsRequest) (*v1alpha1.GetOperationsResponse, error) {
	queryParams := &storage.OperationsQueryParameters{}
	if req.GetService() != "" {
		queryParams.ServiceName = req.Service
	}

	if req.SpanKind != "" {
		queryParams.SpanKind = req.SpanKind
	}

	operations, err := t.QueryService.tracingQuerySvc.GetOperations(ctx, queryParams)
	if err != nil {
		zap.S().Errorf("query operations failed: %s", zap.Error(err).String)
		return nil, err
	}

	return &v1alpha1.GetOperationsResponse{
		Names: operations,
	}, nil
}

// find traces by params
func (t *Handler) SearchTraces(ctx context.Context, request *v1alpha1.FindTracesRequest) (*v1.TracesData, error) {
	queryParams, err := parseTraceQueryParameters(request)
	if err != nil {
		return nil, err
	}

	traces, err := t.QueryService.tracingQuerySvc.SearchTraces(ctx, queryParams)
	if err != nil {
		zap.S().Errorf("query tracing failed: %s", zap.Error(err).String)
		return nil, err
	}

	return traces, nil
}

func (t *Handler) SearchLogs(ctx context.Context, request *v1alpha1.GetLogsRequest) (*v1_logs.LogsData, error) {
	//TODO:
	return nil, nil
}

func (t *Handler) GetTrace(ctx context.Context, request *v1alpha1.GetTraceRequest) (*v1.TracesData, error) {
	trace, err := t.QueryService.tracingQuerySvc.GetTrace(ctx, request.TraceId)
	if err != nil {
		return nil, err
	}

	return &v1.TracesData{
		ResourceSpans: trace.ResourceSpans,
	}, nil
}

func (t *Handler) GetServices(ctx context.Context, _ *v1alpha1.GetServicesRequest) (*v1alpha1.ResourcesData, error) {
	kvs, err := t.QueryService.tracingQuerySvc.GetService(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.ResourcesData{Resources: kvs}, nil
}

func parseTraceQueryParameters(request *v1alpha1.FindTracesRequest) (*storage.TraceQueryParameters, error) {
	q := request.Query
	queryParams := &storage.TraceQueryParameters{}
	if q != nil {
		tags := map[string]string{}

		if q.StartTime != nil {
			queryParams.StartTime = q.StartTime.AsTime()
		}
		if q.EndTime != nil {
			queryParams.EndTime = q.EndTime.AsTime()
		}

		if q.ServiceName != "" {
			queryParams.ServiceName = q.ServiceName
		}

		if q.OperationName != "" {
			queryParams.OperationName = q.OperationName
		}

		if len(q.Attributes) > 0 {
			for k, v := range q.Attributes {
				tags[k] = v
			}
		}
		queryParams.Tags = tags

		if q.DurationMin != nil {
			queryParams.DurationMin = q.DurationMin
		}
		if q.DurationMax != nil {
			queryParams.DurationMax = q.DurationMax
		}

		if q.NumTraces > 0 {
			queryParams.NumTraces = int(q.NumTraces)
		}
	}
	return queryParams, nil
}
