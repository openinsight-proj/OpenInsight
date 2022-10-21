package query

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/tracing/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
)

type Handler struct {
	QueryService *QueryService
}

func (t *Handler) GetOperations(context.Context, *v1alpha1.GetOperationsRequest) (*v1alpha1.GetOperationsResponse, error) {

	//TODO:
	return &v1alpha1.GetOperationsResponse{
		Operations: nil,
	}, nil
}

// find traces by params
func (t *Handler) SearchTraces(ctx context.Context, request *v1alpha1.FindTracesRequest) (*v1.TracesData, error) {
	queryParams, err := parseTraceQueryParameters(request.Query)
	if err != nil {
		return nil, err
	}

	traces, err := t.QueryService.tracingQuerySvc.SearchTraces(ctx, queryParams)
	if err != nil {
		zap.S().Errorf("query tracing failed: %s", zap.Error(err).String)
	}

	return traces, nil
}

func (t *Handler) SearchLogs(ctx context.Context, request *v1alpha1.GetLogsRequest) (*v1_logs.LogsData, error) {
	//TODO:
	return nil, nil
}

func (t *Handler) GetTrace(ctx context.Context, request *v1alpha1.GetTraceRequest) (*v1.TracesData, error) {
	return &v1.TracesData{}, nil
}

func (t *Handler) GetServices(ctx context.Context, _ *v1alpha1.GetServicesRequest) (*v1alpha1.GetServicesResponse, error) {

	return &v1alpha1.GetServicesResponse{}, nil
}

func parseTraceQueryParameters(q *v1alpha1.TraceQueryParameters) (*storage.TraceQueryParameters, error) {
	queryParams := &storage.TraceQueryParameters{}
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
	return queryParams, nil

}
