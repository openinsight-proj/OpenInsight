package handler

import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/datasource"

type QueryService struct {
	//ES client
	// vm client
	TracingQuerySvc datasource.Query
}
