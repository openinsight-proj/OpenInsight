package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type gaugeModel struct {
	MetricName        string
	MetricDescription string
	MetricUnit        string
	gauge             pmetric.Gauge
}

type GaugeMetrics struct {
	GaugeModels []*gaugeModel
	Metadata    *MetricsMetaData
	InsertSQL   string
}

func (g *GaugeMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	start := time.Now()
	//do insert
	statement, err := tx.PrepareContext(ctx, g.InsertSQL)
	if err != nil {
		return fmt.Errorf("PrepareContext:%w", err)
	}
	defer func() {
		_ = statement.Close()
	}()
	count := 0
	for _, model := range g.GaugeModels {
		for i := 0; i < model.gauge.DataPoints().Len(); i++ {
			dp := model.gauge.DataPoints().At(i)
			attrs, times, floatValues, intValues, traceIDs, spanIDs := convertExemplars(dp.Exemplars())
			//todo 区分 ValueAsDouble ValueAsInt
			_, err = statement.ExecContext(ctx,
				g.Metadata.ResAttr,
				g.Metadata.ResUrl,
				g.Metadata.ScopeInstr.Name(),
				g.Metadata.ScopeInstr.Version(),
				g.Metadata.ScopeInstr.Attributes(),
				g.Metadata.ScopeUrl,
				model.MetricName,
				model.MetricDescription,
				model.MetricUnit,
				attributesToMap(dp.Attributes()),
				dp.Timestamp().AsTime(),
				dp.DoubleValue(),
				dp.IntValue(),
				dp.Flags(),
				attrs,
				times,
				floatValues,
				intValues,
				traceIDs,
				spanIDs,
			)
			if err != nil {
				return fmt.Errorf("ExecContext:%w", err)
			}
			count++
		}
	}

	duration := time.Since(start)
	logger.Info("insert gauge metrics", zap.Int("records", count),
		zap.String("cost", duration.String()))
	return nil
}

func (g *GaugeMetrics) Add(metrics interface{}, name string, description string, unit string) {
	gauge, _ := metrics.(pmetric.Gauge)
	g.GaugeModels = append(g.GaugeModels, &gaugeModel{
		MetricName:        name,
		MetricDescription: description,
		MetricUnit:        unit,
		gauge:             gauge,
	})
}

func (g *GaugeMetrics) InjectMetaData(Metadata *MetricsMetaData) {
	g.Metadata = Metadata
}
