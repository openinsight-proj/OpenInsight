package datasource

import (
	"go.uber.org/zap"
)

type Factory interface {
	Initialize(logger *zap.Logger) error
	// CreateSpanQuery creates a datasource.Query.
	CreateSpanQuery() (Query, error)
}
