package query

import (
	"context"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"net/http"
)

type queryServer struct {
	config Config
	logger *zap.Logger
	server *http.Server
	stopCh chan struct{}
	//TODO: http/grpc settings
	settings component.TelemetrySettings
}

var _ component.PipelineWatcher = (*queryServer)(nil)

func (qs *queryServer) Start(_ context.Context, host component.Host) error {
	return nil
}

func (qs *queryServer) Shutdown(context.Context) error {
	return nil
}

func (qs *queryServer) Ready() error {
	return nil
}

func (qs *queryServer) NotReady() error {
	return nil
}

func newQueryServer(config Config, settings component.TelemetrySettings) *queryServer {
	qs := &queryServer{
		config:   config,
		logger:   settings.Logger,
		settings: settings,
	}

	// init grpc-gateway
	// init grpc server

	return qs
}
