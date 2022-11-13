package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const gaugePlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

type gaugeModel struct {
	metricName        string
	metricDescription string
	metricUnit        string
	gauge             pmetric.Gauge
}

type GaugeMetrics struct {
	gaugeModels []*gaugeModel
	metadata    *MetricsMetaData
	InsertSQL   string
}

func (g *GaugeMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	start := time.Now()
	//do insert
	//statement, err := tx.PrepareContext(ctx, g.InsertSQL)
	//if err != nil {
	//	return fmt.Errorf("PrepareContext:%w", err)
	//}
	//defer func() {
	//	_ = statement.Close()
	//}()

	//ecec
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range g.gaugeModels {
		for i := 0; i < model.gauge.DataPoints().Len(); i++ {
			dp := model.gauge.DataPoints().At(i)
			//todo distinguish ValueAsDouble ValueAsInt
			valuePlaceholders = append(valuePlaceholders, gaugePlaceholders)
			valueArgs = append(valueArgs, g.metadata.ResAttr)
			valueArgs = append(valueArgs, g.metadata.ResUrl)
			valueArgs = append(valueArgs, g.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, g.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(g.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, g.metadata.ScopeUrl)
			valueArgs = append(valueArgs, model.metricName)
			valueArgs = append(valueArgs, model.metricDescription)
			valueArgs = append(valueArgs, model.metricUnit)
			valueArgs = append(valueArgs, attributesToMap(dp.Attributes()))
			valueArgs = append(valueArgs, dp.Timestamp().AsTime())
			valueArgs = append(valueArgs, dp.DoubleValue())
			valueArgs = append(valueArgs, dp.IntValue())
			valueArgs = append(valueArgs, uint32(dp.Flags()))

			//attrs, times, floatValues, intValues, traceIDs, spanIDs := convertExemplars(dp.Exemplars())
			//valueArgs = append(valueArgs, attrs)
			//valueArgs = append(valueArgs, times)
			//valueArgs = append(valueArgs, floatValues)
			//valueArgs = append(valueArgs, intValues)
			//valueArgs = append(valueArgs, traceIDs)
			//valueArgs = append(valueArgs, spanIDs)
			//_, err = statement.ExecContext(ctx,
			//	g.metadata.ResAttr,
			//	g.metadata.ResUrl,
			//	g.metadata.ScopeInstr.Name(),
			//	g.metadata.ScopeInstr.Version(),
			//	attributesToMap(g.metadata.ScopeInstr.Attributes()),
			//	g.metadata.ScopeUrl,
			//	model.metricName,
			//	model.metricDescription,
			//	model.metricUnit,
			//	attributesToMap(dp.Attributes()),
			//	dp.Timestamp().AsTime(),
			//	dp.DoubleValue(),
			//	dp.IntValue(),
			//	uint32(dp.Flags()),
			//	attrs,
			//	times,
			//	floatValues,
			//	intValues,
			//	traceIDs,
			//	spanIDs,
			//)
			//if err != nil {
			//	return fmt.Errorf("ExecContext:%w", err)
			//}
		}
	}
	query := fmt.Sprintf("%s %s", g.InsertSQL, strings.Join(valuePlaceholders, ","))
	_, err := tx.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("ExecContext:%w", err)
	}
	duration := time.Since(start)

	logger.Info("insert gauge metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return errors.New("raise an error")
	//return nil
}

func (g *GaugeMetrics) Add(metrics interface{}, name string, description string, unit string) {
	gauge, _ := metrics.(pmetric.Gauge)
	g.gaugeModels = append(g.gaugeModels, &gaugeModel{
		metricName:        name,
		metricDescription: description,
		metricUnit:        unit,
		gauge:             gauge,
	})
}

func (g *GaugeMetrics) InjectMetaData(metadata *MetricsMetaData) {
	g.metadata = metadata
}
