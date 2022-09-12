package query

import (
	"context"
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/tracing/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"go.uber.org/zap"
)

const (
	defaultQueryLimit = 100

	serviceParam               = "service"
	tracingK8sNamespaceNameTag = "k8s.namespace.name"
	tracingK8sClusterUIdTag    = "k8s.cluster.id"
)

var (
	errServiceParameterRequired = fmt.Errorf("parameter '%s' is required", serviceParam)
	errParsTime                 = fmt.Errorf("start or end time must be 2006-01-02T15:04:05.000Z like string")
	errParsDuration             = fmt.Errorf("duration must be like 10ns 300ms or 1m")
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
func (t *Handler) FindTraces(ctx context.Context, request *v1alpha1.FindTracesRequest) (*v1alpha1.SpansResponseChunk, error) {

	_, err := t.QueryService.tracingQuerySvc.FindTraces(ctx, &storage.TraceQueryParameters{})
	if err != nil {
		zap.S().Errorf("query tracing failed: $s", zap.Error(err))
	}
	return &v1alpha1.SpansResponseChunk{}, nil
}

func (t *Handler) GetTrace(ctx context.Context, request *v1alpha1.GetTraceRequest) (*v1alpha1.SpansResponseChunk, error) {
	return &v1alpha1.SpansResponseChunk{}, nil
}

func (t *Handler) GetServices(ctx context.Context, _ *v1alpha1.GetServicesRequest) (*v1alpha1.GetServicesResponse, error) {

	return &v1alpha1.GetServicesResponse{}, nil
}
