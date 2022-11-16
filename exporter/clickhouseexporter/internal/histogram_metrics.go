package internal

import (
	"context"
	"database/sql"
	"fmt"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"strings"
	"time"
)

const histogramPlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

type histogramModel struct {
	metricName        string
	metricDescription string
	metricUnit        string
	histogram         pmetric.Histogram
}

type HistogramMetrics struct {
	histogramModel []*histogramModel
	metadata       *MetricsMetaData
	InsertSQL      string
}

func (h *HistogramMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range h.histogramModel {
		for i := 0; i < model.histogram.DataPoints().Len(); i++ {
			dp := model.histogram.DataPoints().At(i)
			valuePlaceholders = append(valuePlaceholders, histogramPlaceholders)

			valueArgs = append(valueArgs, h.metadata.ResAttr)
			valueArgs = append(valueArgs, h.metadata.ResUrl)
			valueArgs = append(valueArgs, h.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, h.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(h.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, h.metadata.ScopeInstr.DroppedAttributesCount())
			valueArgs = append(valueArgs, h.metadata.ScopeUrl)
			valueArgs = append(valueArgs, model.metricName)
			valueArgs = append(valueArgs, model.metricDescription)
			valueArgs = append(valueArgs, model.metricUnit)
			valueArgs = append(valueArgs, attributesToMap(dp.Attributes()))
			valueArgs = append(valueArgs, dp.StartTimestamp().AsTime())
			valueArgs = append(valueArgs, dp.Timestamp().AsTime())
			valueArgs = append(valueArgs, dp.Count())
			valueArgs = append(valueArgs, dp.Sum())
			valueArgs = append(valueArgs, convertSliceToArraySet(dp.BucketCounts().AsRaw()))
			valueArgs = append(valueArgs, convertSliceToArraySet(dp.ExplicitBounds().AsRaw()))

			attrs, times, floatValues, intValues, traceIDs, spanIDs := convertExemplars(dp.Exemplars())
			valueArgs = append(valueArgs, attrs)
			valueArgs = append(valueArgs, times)
			valueArgs = append(valueArgs, floatValues)
			valueArgs = append(valueArgs, intValues)
			valueArgs = append(valueArgs, traceIDs)
			valueArgs = append(valueArgs, spanIDs)
			valueArgs = append(valueArgs, uint32(dp.Flags()))
			valueArgs = append(valueArgs, dp.Min())
			valueArgs = append(valueArgs, dp.Max())
		}
	}

	if len(valuePlaceholders) == 0 {
		return nil
	}

	start := time.Now()
	query := fmt.Sprintf("%s %s", h.InsertSQL, strings.Join(valuePlaceholders, ","))
	_, err := tx.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("insert histogram metrics fail:%w", err)
	}
	duration := time.Since(start)

	//TODO latency metrics
	logger.Info("insert histogram metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return nil
}

func (h *HistogramMetrics) Add(metrics interface{}, name string, description string, unit string) {
	histogram, _ := metrics.(pmetric.Histogram)
	h.histogramModel = append(h.histogramModel, &histogramModel{
		metricName:        name,
		metricDescription: description,
		metricUnit:        unit,
		histogram:         histogram,
	})
}

func (h *HistogramMetrics) InjectMetaData(metaData *MetricsMetaData) {
	h.metadata = metaData
}
