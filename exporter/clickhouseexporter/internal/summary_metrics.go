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

const summaryPlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

type summaryModel struct {
	metricName        string
	metricDescription string
	metricUnit        string
	summary           pmetric.Summary
}

type SummaryMetrics struct {
	summaryModel []*summaryModel
	metadata     *MetricsMetaData
	InsertSQL    string
}

func (s *SummaryMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range s.summaryModel {
		for i := 0; i < model.summary.DataPoints().Len(); i++ {
			dp := model.summary.DataPoints().At(i)
			valuePlaceholders = append(valuePlaceholders, summaryPlaceholders)

			valueArgs = append(valueArgs, s.metadata.ResAttr)
			valueArgs = append(valueArgs, s.metadata.ResURL)
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(s.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, s.metadata.ScopeInstr.DroppedAttributesCount())
			valueArgs = append(valueArgs, s.metadata.ScopeURL)
			valueArgs = append(valueArgs, model.metricName)
			valueArgs = append(valueArgs, model.metricDescription)
			valueArgs = append(valueArgs, model.metricUnit)
			valueArgs = append(valueArgs, attributesToMap(dp.Attributes()))
			valueArgs = append(valueArgs, dp.StartTimestamp().AsTime().UnixNano())
			valueArgs = append(valueArgs, dp.Timestamp().AsTime().UnixNano())
			valueArgs = append(valueArgs, dp.Count())
			valueArgs = append(valueArgs, dp.Sum())

			quantiles, values := convertValueAtQuantile(dp.QuantileValues())
			valueArgs = append(valueArgs, quantiles)
			valueArgs = append(valueArgs, values)
			valueArgs = append(valueArgs, uint32(dp.Flags()))
		}
	}

	if len(valuePlaceholders) == 0 {
		return nil
	}

	start := time.Now()
	_, err := tx.ExecContext(ctx, fmt.Sprintf("%s %s", s.InsertSQL, strings.Join(valuePlaceholders, ",")), valueArgs...)
	if err != nil {
		return fmt.Errorf("insert summary metrics fail:%w", err)
	}
	duration := time.Since(start)

	// TODO latency metrics
	logger.Info("insert summary metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return nil
}

func (s *SummaryMetrics) Add(metrics interface{}, name string, description string, unit string) {
	summary, _ := metrics.(pmetric.Summary)
	s.summaryModel = append(s.summaryModel, &summaryModel{
		metricName:        name,
		metricDescription: description,
		metricUnit:        unit,
		summary:           summary,
	})
}

func (s *SummaryMetrics) InjectMetaData(metaData *MetricsMetaData) {
	s.metadata = metaData
}
