package storage

import (
	"go.uber.org/zap"
)

type Factory interface {
	Initialize(logger *zap.Logger) error
	// CreateSpanQuery creates a storage.Query.
	CreateSpanQuery() (Query, error)
}
