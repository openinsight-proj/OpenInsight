// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter/internal"

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const sumPlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

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
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range s.sumModel {
		for i := 0; i < model.sum.DataPoints().Len(); i++ {
			dp := model.sum.DataPoints().At(i)
			valuePlaceholders = append(valuePlaceholders, sumPlaceholders)

			valueArgs = append(valueArgs, s.metadata.ResAttr)
			valueArgs = append(valueArgs, s.metadata.ResURL)
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(s.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.DroppedAttributesCount())
			valueArgs = append(valueArgs, s.metadata.ScopeURL)
			valueArgs = append(valueArgs, s.metadata.ServiceName)
			valueArgs = append(valueArgs, model.metricName)
			valueArgs = append(valueArgs, model.metricDescription)
			valueArgs = append(valueArgs, model.metricUnit)
			valueArgs = append(valueArgs, attributesToMap(dp.Attributes()))
			valueArgs = append(valueArgs, dp.Timestamp().AsTime())
			valueArgs = append(valueArgs, dp.DoubleValue())
			valueArgs = append(valueArgs, dp.IntValue())
			valueArgs = append(valueArgs, uint32(dp.Flags()))

			attrs, times, floatValues, intValues, traceIDs, spanIDs := convertExemplars(dp.Exemplars())
			valueArgs = append(valueArgs, attrs)
			valueArgs = append(valueArgs, times)
			valueArgs = append(valueArgs, floatValues)
			valueArgs = append(valueArgs, intValues)
			valueArgs = append(valueArgs, traceIDs)
			valueArgs = append(valueArgs, spanIDs)
			valueArgs = append(valueArgs, int32(model.sum.AggregationTemporality()))
			valueArgs = append(valueArgs, model.sum.IsMonotonic())
		}
	}

	if len(valuePlaceholders) == 0 {
		return nil
	}

	start := time.Now()
	_, err := tx.ExecContext(ctx, fmt.Sprintf("%s %s", s.InsertSQL, strings.Join(valuePlaceholders, ",")), valueArgs...)
	if err != nil {
		return fmt.Errorf("insert sum metrics fail:%w", err)
	}
	duration := time.Since(start)

	// TODO latency metrics
	logger.Info("insert sum metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return nil
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
