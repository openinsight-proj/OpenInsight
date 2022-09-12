package query

import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/plugin/storage"

type QueryService struct {
	//ES client
	// vm client
	tracingQuerySvc storage.Query
}
