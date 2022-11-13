package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"strings"
	"time"
)

const sumPlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

type sumModel struct {
	metricName        string
	metricDescription string
	metricUnit        string
	sum               pmetric.Sum
}

type SumMetrics struct {
	sumModel  []*sumModel
	metadata  *MetricsMetaData
	InsertSQL string
}

func (s *SumMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	start := time.Now()
	//do insert
	//statement, err := tx.PrepareContext(ctx, s.InsertSQL)
	//if err != nil {
	//	return fmt.Errorf("PrepareContext:%w", err)
	//}
	//defer func() {
	//	_ = statement.Close()
	//}()

	//ecec
	var valuePlaceholders []string
	var valueArgs []interface{}
	for _, model := range s.sumModel {
		for i := 0; i < model.sum.DataPoints().Len(); i++ {
			dp := model.sum.DataPoints().At(i)

			//todo distinguish ValueAsDouble ValueAsInt
			valuePlaceholders = append(valuePlaceholders, sumPlaceholders)
			valueArgs = append(valueArgs, s.metadata.ResAttr)
			valueArgs = append(valueArgs, s.metadata.ResUrl)
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(s.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, s.metadata.ScopeUrl)
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
			valueArgs = append(valueArgs, int32(model.sum.AggregationTemporality()))
			valueArgs = append(valueArgs, model.sum.IsMonotonic())
			//todo distinguish ValueAsDouble ValueAsInt
			//_, err = statement.ExecContext(ctx,
			//	s.metadata.ResAttr,
			//	s.metadata.ResUrl,
			//	s.metadata.ScopeInstr.Name(),
			//	s.metadata.ScopeInstr.Version(),
			//	attributesToMap(s.metadata.ScopeInstr.Attributes()),
			//	s.metadata.ScopeUrl,
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
			//	int32(model.sum.AggregationTemporality()),
			//	model.sum.IsMonotonic(),
			//)
			//if err != nil {
			//	return fmt.Errorf("ExecContext:%w", err)
			//}
		}
	}
	query := fmt.Sprintf("%s %s", s.InsertSQL, strings.Join(valuePlaceholders, ","))
	_, err := tx.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("ExecContext:%w", err)
	}
	duration := time.Since(start)
	logger.Info("insert gauge metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	//return nil
	return errors.New("raise an error")
}

func (s *SumMetrics) Add(metrics interface{}, name string, description string, unit string) {
	sum, _ := metrics.(pmetric.Sum)
	s.sumModel = append(s.sumModel, &sumModel{
		metricName:        name,
		metricDescription: description,
		metricUnit:        unit,
		sum:               sum,
	})
}

func (s *SumMetrics) InjectMetaData(metadata *MetricsMetaData) {
	s.metadata = metadata
}
