package query

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/handler"
	"go.opentelemetry.io/collector/extension"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// max msg size 20M
const maxMsgSize = 20 * 1024 * 1024

type queryServer struct {
	config           *Config
	logger           *zap.Logger
	cmux             cmux.CMux
	router           *mux.Router
	httpServer       *http.Server
	grpcServer       *grpc.Server
	grpcConn         net.Listener
	httpConn         net.Listener
	GatewayServerMux *runtime.ServeMux
	settings         component.TelemetrySettings
}

var _ extension.PipelineWatcher = (*queryServer)(nil)

func (qs *queryServer) Start(_ context.Context, host component.Host) error {
	closeGRPCGateway, err := qs.Server()
	if err != nil {
		closeGRPCGateway()
		return err
	}

	go func() {
		err = qs.grpcServer.Serve(qs.grpcConn)
		if err != nil {
			zap.S().Fatalf("grpc listener failed: %v", err)
		}
	}()

	go func() {
		err = qs.httpServer.Serve(qs.httpConn)
		if err != nil {
			zap.S().Fatalf("http listener failed: %v", err)
		}
	}()

	go func() {
		err = qs.cmux.Serve()
		if err != nil {
			zap.S().Fatalf("grpc gateway cmux listener failed: %v", err)
		}
	}()
	return nil
}

func (qs *queryServer) Shutdown(context.Context) error {
	qs.grpcServer.Stop()
	err := qs.httpServer.Close()
	if err != nil {
		return err
	}
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

func (qs *queryServer) initFactories() (*plugin.Factory, error) {
	factories, err := plugin.NewFactory(&plugin.FactoryConfig{
		ElasticsearchStorage: qs.config.Storage.ElasticsearchType,
		ClickhouseStorage:    qs.config.Storage.ClickhouseType,
		TracingQuery:         qs.config.TracingQuery,
		MetricsQuery:         qs.config.MetricsQuery,
		LoggingQuery:         qs.config.LoggingQuery,
	})
	if err != nil {
		qs.logger.Fatal("Failed init factories", zap.Error(err))
		return nil, err
	}
	if err := factories.Initialize(qs.logger); err != nil {
		qs.logger.Fatal("Failed to init storage factory", zap.Error(err))
		return nil, err
	}
	return factories, nil
}

func (qs *queryServer) Server() (context.CancelFunc, error) {
	err := qs.initListener()
	if err != nil {
		return nil, err
	}

	marshaller := &runtime.JSONPb{}
	marshaller.UseProtoNames = false
	marshaller.EmitUnpopulated = true
	qs.GatewayServerMux = runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, marshaller))

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	extendMsgSizeOpt := grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize))
	extraOpt := append(opts, extendMsgSizeOpt)

	ctx, closeGRPCGateway := context.WithCancel(context.Background())

	factories, err := qs.initFactories()
	if err != nil {
		return closeGRPCGateway, err
	}
	tracingQuerySvc, err := factories.CreateSpanQuery()
	if err != nil {
		qs.logger.Fatal("Failed to create span reader", zap.Error(err))
	}

	qSvc := &handler.QueryService{
		TracingQuerySvc: tracingQuerySvc,
	}

	v1alpha1.RegisterQueryServiceServer(qs.grpcServer, &handler.Handler{QueryService: qSvc})
	err = v1alpha1.RegisterQueryServiceHandlerFromEndpoint(ctx, qs.GatewayServerMux, qs.config.Protocols.Http.Endpoint, extraOpt)
	if err != nil {
		closeGRPCGateway()
		return closeGRPCGateway, err
	}

	qs.router = mux.NewRouter()
	qs.router.PathPrefix("/").Handler(qs.GatewayServerMux)
	qs.httpServer.Handler = qs.router
	qs.cmux = cmux.New(qs.httpConn)
	qs.grpcConn = qs.cmux.Match(cmux.HTTP2())
	qs.httpConn = qs.cmux.Match(cmux.Any())
	return closeGRPCGateway, err
}

func (qs *queryServer) initListener() error {
	// Create protocol servers
	qs.grpcServer = grpc.NewServer(grpc.MaxSendMsgSize(maxMsgSize))
	qs.httpServer = &http.Server{Addr: qs.config.Http.Endpoint}

	var err error
	qs.grpcConn, err = qs.config.Grpc.NetAddr.Listen()
	if err != nil {
		return err
	}

	qs.httpConn, err = qs.config.Http.ToListener()
	if err != nil {
		return err
	}
	return nil
}
