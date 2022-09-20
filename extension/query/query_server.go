package query

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/tracing/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
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
	closeGRPCGateway, err := qs.Server()
	if err != nil {
		closeGRPCGateway()
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

func (s *queryServer) Server() (context.CancelFunc, error) {
	listener, err := s.config.Protocols.Http.ToListener()
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %s: %w", s.config.Protocols.Http.Endpoint, err)
	}
	s.cmux = cmux.New(listener)
	grpcListener := s.cmux.Match(cmux.HTTP2())
	httpListener := s.cmux.Match(cmux.HTTP1Fast())

	// Create protocol servers
	s.grpcServer = grpc.NewServer(grpc.MaxSendMsgSize(maxMsgSize))
	s.httpServer = &http.Server{Addr: s.config.Http.Endpoint}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	extendMsgSizeOpt := grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize))
	extraOpt := append(opts, extendMsgSizeOpt)

	marshaller := &runtime.JSONPb{}
	marshaller.UseProtoNames = false
	marshaller.EmitUnpopulated = true
	s.GatewayServerMux = runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, marshaller))

	ctx, closeGRPCGateway := context.WithCancel(context.Background())

	// TODO: register through factories.
	factories, err := plugin.NewFactory(&plugin.FactoryConfig{
		TracingQuery: s.config.TracingQuery,
		MetricsQuery: s.config.MetricsQuery,
		LoggingQuery: s.config.LoggingQuery,
	})

	if err != nil {
		s.logger.Fatal("Failed init factories", zap.Error(err))
	}
	if err := factories.Initialize(s.logger); err != nil {
		zap.S().Fatal("Failed to init storage factory", zap.Error(err))
	}

	tracingQuerySvc, err := factories.CreateSpanQuery()
	if err != nil {
		zap.S().Fatal("Failed to create span reader", zap.Error(err))
	}

	v1alpha1.RegisterQueryServiceServer(s.grpcServer, &Handler{QueryService: &QueryService{
		tracingQuerySvc: tracingQuerySvc,
	}})
	err = v1alpha1.RegisterQueryServiceHandlerFromEndpoint(ctx, s.GatewayServerMux, s.config.Protocols.Http.Endpoint, extraOpt)
	if err != nil {
		closeGRPCGateway()
		return closeGRPCGateway, err
	}

	s.router = mux.NewRouter().UseEncodedPath()
	s.router.PathPrefix("/").Handler(s.GatewayServerMux)
	s.httpServer.Handler = s.router

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
	return closeGRPCGateway, err
}
