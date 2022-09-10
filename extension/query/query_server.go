package query

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/tracing/v1alpha1"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
)

// max msg size 20M
const maxMsgSize = 20 * 1024 * 1024

type queryServer struct {
	config *Config
	logger *zap.Logger
	server *http.Server
	stopCh chan struct{}
	//TODO: http/grpc settings
	cmux             cmux.CMux
	router           *mux.Router
	httpServer       *http.Server
	grpcServer       *grpc.Server
	GatewayServerMux *runtime.ServeMux
	settings         component.TelemetrySettings
}

var _ component.PipelineWatcher = (*queryServer)(nil)

func (qs *queryServer) Start(_ context.Context, host component.Host) error {
	err := qs.Server()
	if err != nil {
		return err
	}
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

func NewQueryServer(config *Config, settings component.TelemetrySettings) *queryServer {
	qs := &queryServer{
		config:   config,
		logger:   settings.Logger,
		settings: settings,
	}
	return qs
}

func (s *queryServer) Server() error {
	listener, err := s.config.Protocols.Http.ToListener()
	if err != nil {
		return fmt.Errorf("failed to bind to address %s: %w", s.config.Protocols.Http.Endpoint, err)
	}
	s.cmux = cmux.New(listener)
	grpcListener := s.cmux.Match(cmux.HTTP2())
	httpListener := s.cmux.Match(cmux.HTTP1Fast())

	// Create protocol servers
	s.grpcServer = grpc.NewServer(grpc.MaxSendMsgSize(maxMsgSize))
	s.httpServer = &http.Server{
		Addr: fmt.Sprintf("bind to address %s", s.config.Http.Endpoint),
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	extendMsgSizeOpt := grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize))
	extraOpt := append(opts, extendMsgSizeOpt)

	marshaller := &runtime.JSONPb{}
	marshaller.UseProtoNames = false
	marshaller.EmitUnpopulated = true
	s.GatewayServerMux = runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, marshaller))

	v1alpha1.RegisterQueryServiceServer(s.grpcServer, &Handler{QueryService: &QueryService{}})
	err = v1alpha1.RegisterQueryServiceHandlerFromEndpoint(context.TODO(), s.GatewayServerMux, s.httpServer.Addr, extraOpt)
	if err != nil {
		return err
	}

	// Use the muxed listeners for your servers.
	go func() {
		err = s.grpcServer.Serve(grpcListener)
		if err != nil {
			zap.S().Fatalf("grpc listener failed: %v", err)
		}
	}()

	go func() {
		err = s.httpServer.Serve(httpListener)
		if err != nil {
			zap.S().Fatalf("http listener failed: %v", err)
		}
	}()

	// Start serve
	err = s.cmux.Serve()
	return err
}
