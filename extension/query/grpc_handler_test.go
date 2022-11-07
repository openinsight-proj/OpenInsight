package query

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/api/tracing/v1alpha1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configtls"
	v1_logs "go.opentelemetry.io/proto/otlp/logs/v1"
	v1_trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"testing"
	"time"
)

type grpcServer struct {
	server  *grpc.Server
	lisAddr net.Addr
}

func Test_server(t *testing.T) {
	lis, _ := net.Listen("tcp", ":8000")
	grpcServer := grpc.NewServer()

	//ck
	ckFactor := clickhouse.NewFactory(&clickhouse.ClickhouseType{
		Dsn: "tcp://10.6.229.191:32022/otel",
		TlsClientSetting: configtls.TLSClientSetting{
			TLSSetting:         configtls.TLSSetting{},
			Insecure:           true,
			InsecureSkipVerify: false,
			ServerName:         "",
		},
		LoggingTableName: "otel_logs",
		TracingTableName: "otel_traces",
		MetricsTableName: "otel_metrics",
	})

	err := ckFactor.Initialize(&zap.Logger{})
	require.NoError(t, err)

	spanQuery, err := ckFactor.CreateSpanQuery()
	require.NoError(t, err)

	handler := &Handler{
		QueryService: &QueryService{
			tracingQuerySvc: spanQuery,
		},
	}

	v1alpha1.RegisterQueryServiceServer(grpcServer, handler)
	err = grpcServer.Serve(lis)
	require.NoError(t, err)
}

func newMockGRPCServer(t *testing.T) (*grpc.Server, net.Addr) {
	lis, _ := net.Listen("tcp", ":8000")
	grpcServer := grpc.NewServer()

	//ck
	ckFactor := clickhouse.NewFactory(&clickhouse.ClickhouseType{
		Dsn: "tcp://10.6.229.191:32022/otel",
		TlsClientSetting: configtls.TLSClientSetting{
			TLSSetting:         configtls.TLSSetting{},
			Insecure:           true,
			InsecureSkipVerify: false,
			ServerName:         "",
		},
		LoggingTableName: "otel_logs",
		TracingTableName: "otel_traces",
		MetricsTableName: "otel_metrics",
	})

	err := ckFactor.Initialize(&zap.Logger{})
	if err != nil {
		return nil, nil
	}
	spanQuery, err := ckFactor.CreateSpanQuery()
	if err != nil {
		return nil, nil
	}

	handler := &Handler{
		QueryService: &QueryService{
			tracingQuerySvc: spanQuery,
		},
	}
	v1alpha1.RegisterQueryServiceServer(grpcServer, handler)

	go func() {
		err := grpcServer.Serve(lis)
		require.NoError(t, err)
	}()

	return grpcServer, lis.Addr()
}

type grpcClient struct {
	conn *grpc.ClientConn
	mock.Mock
}

func newGRPCClient(t *testing.T, addr string) *grpcClient {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return &grpcClient{
		conn: conn,
	}
}

func initializeTestServerGRPCWithOptions(t *testing.T) *grpcServer {
	server, addr := newMockGRPCServer(t)
	return &grpcServer{
		server:  server,
		lisAddr: addr,
	}
}

func withServerAndClient(t *testing.T, actualTest func(server *grpcServer, client *grpcClient)) {
	server := initializeTestServerGRPCWithOptions(t)
	client := newGRPCClient(t, server.lisAddr.String())
	defer server.server.Stop()
	defer client.conn.Close()

	actualTest(server, client)
}

func Test_ParseTraceQueryParameters(t *testing.T) {
	tests := []struct {
		caseStr  string
		request  *v1alpha1.TraceQueryParameters
		expected *storage.TraceQueryParameters
	}{
		{
			caseStr: "only serviceName",
			request: &v1alpha1.TraceQueryParameters{
				ServiceName: "foo",
			},
			expected: &storage.TraceQueryParameters{
				ServiceName: "foo",
			},
		},
		{
			caseStr: "only start time",
			request: &v1alpha1.TraceQueryParameters{
				StartTime: timestamppb.New(time.Unix(1666331554, 0).UTC()),
			},
			expected: &storage.TraceQueryParameters{
				StartTime: time.Unix(1666331554, 0).UTC(),
			},
		},
		{
			caseStr: "only attributes",
			request: &v1alpha1.TraceQueryParameters{
				Attributes: map[string]string{
					"foo":   "bar",
					"hello": "world",
				},
			},
			expected: &storage.TraceQueryParameters{
				Tags: map[string]string{
					"foo":   "bar",
					"hello": "world",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.caseStr, func(t *testing.T) {
			//actual, err := parseTraceQueryParameters(tc)
			//require.NoError(t, err)
			//assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSearchTraces(t *testing.T) {
	withServerAndClient(t, func(server *grpcServer, client *grpcClient) {

		handler := &Handler{
			QueryService: &QueryService{
				tracingQuerySvc: &MockQuery{},
			},
		}

		req := &v1alpha1.FindTracesRequest{
			Query: &v1alpha1.TraceQueryParameters{
				ServiceName:   "",
				OperationName: "",
				Attributes:    nil,
				StartTime:     nil,
				EndTime:       nil,
				DurationMin:   nil,
				DurationMax:   nil,
				NumTraces:     0,
			},
		}
		tracesData, err := handler.SearchTraces(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, 0, len(tracesData.ResourceSpans))
	})
}

type MockQuery struct {
}

func (m *MockQuery) SearchTraces(ctx context.Context, query *storage.TraceQueryParameters) (*v1_trace.TracesData, error) {
	return nil, nil
}

func (m *MockQuery) SearchLogs(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (m *MockQuery) GetTrace(ctx context.Context, traceID string) (*v1_trace.TracesData, error) {
	return nil, nil
}

func (m *MockQuery) GetServices(ctx context.Context, _ *v1alpha1.GetServicesRequest) (*v1alpha1.GetServicesResponse, error) {
	return nil, nil
}

func (m *MockQuery) GetLog(ctx context.Context) (*v1_logs.LogsData, error) {
	return nil, nil
}

func (m *MockQuery) GetService(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *MockQuery) GetOperations(ctx context.Context, query *storage.OperationsQueryParameters) ([]string, error) {
	return nil, nil
}
