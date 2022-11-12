package internal

import (
	"context"
	"database/sql"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type MetricsModel interface {
	Add(metrics interface{}, name string, description string, unit string)
	InjectMetaData(metaData *MetricsMetaData)
	Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error
}

type MetricsMetaData struct {
	ResAttr    map[string]string
	ResUrl     string
	ScopeUrl   string
	ScopeInstr pcommon.InstrumentationScope
}

func convertExemplars(exemplars pmetric.ExemplarSlice) ([]map[string]string, []time.Time, []float64, []int64, []string, []string) {
	var (
		attrs       []map[string]string
		times       []time.Time
		floatValues []float64
		intValues   []int64
		traceIDs    []string
		spanIDs     []string
	)
	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		attrs = append(attrs, attributesToMap(exemplar.FilteredAttributes()))
		times = append(times, exemplar.Timestamp().AsTime())
		floatValues = append(floatValues, exemplar.DoubleValue())
		intValues = append(intValues, exemplar.IntValue())
		traceIDs = append(traceIDs, exemplar.TraceID().HexString())
		spanIDs = append(spanIDs, exemplar.SpanID().HexString())
	}
	return attrs, times, floatValues, intValues, traceIDs, spanIDs
}

func attributesToMap(attributes pcommon.Map) map[string]string {
	m := make(map[string]string, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		m[k] = v.Str()
		return true
	})
	return m
}
