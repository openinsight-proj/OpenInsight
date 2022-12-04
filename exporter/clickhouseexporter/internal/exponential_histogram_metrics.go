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

const expHistogramPlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

type expHistogramModel struct {
	metricName        string
	metricDescription string
	metricUnit        string
	expHistogram      pmetric.ExponentialHistogram
}

type ExpHistogramMetrics struct {
	expHistogramModel []*expHistogramModel
	metadata          *MetricsMetaData
	InsertSQL         string
}

func (e *ExpHistogramMetrics) Insert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range e.expHistogramModel {
		for i := 0; i < model.expHistogram.DataPoints().Len(); i++ {
			dp := model.expHistogram.DataPoints().At(i)
			valuePlaceholders = append(valuePlaceholders, expHistogramPlaceholders)

			valueArgs = append(valueArgs, e.metadata.ResAttr)
			valueArgs = append(valueArgs, e.metadata.ResURL)
			valueArgs = append(valueArgs, e.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, e.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(e.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, e.metadata.ScopeInstr.DroppedAttributesCount())
			valueArgs = append(valueArgs, e.metadata.ScopeURL)
			valueArgs = append(valueArgs, e.metadata.ServiceName)
			valueArgs = append(valueArgs, model.metricName)
			valueArgs = append(valueArgs, model.metricDescription)
			valueArgs = append(valueArgs, model.metricUnit)
			valueArgs = append(valueArgs, attributesToMap(dp.Attributes()))
			valueArgs = append(valueArgs, dp.StartTimestamp().AsTime())
			valueArgs = append(valueArgs, dp.Timestamp().AsTime())
			valueArgs = append(valueArgs, dp.Count())
			valueArgs = append(valueArgs, dp.Sum())
			valueArgs = append(valueArgs, dp.Scale())
			valueArgs = append(valueArgs, dp.ZeroCount())
			valueArgs = append(valueArgs, dp.Positive().Offset())
			valueArgs = append(valueArgs, convertSliceToArraySet(dp.Positive().BucketCounts().AsRaw()))
			valueArgs = append(valueArgs, dp.Negative().Offset())
			valueArgs = append(valueArgs, convertSliceToArraySet(dp.Negative().BucketCounts().AsRaw()))

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
	_, err := tx.ExecContext(ctx, fmt.Sprintf("%s %s", e.InsertSQL, strings.Join(valuePlaceholders, ",")), valueArgs...)
	if err != nil {
		return fmt.Errorf("insert exponential histogram metrics fail:%w", err)
	}
	duration := time.Since(start)

	// TODO latency metrics
	logger.Info("insert exponential histogram metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return nil
}

func (e *ExpHistogramMetrics) Add(metrics interface{}, name string, description string, unit string) {
	expHistogram, _ := metrics.(pmetric.ExponentialHistogram)
	e.expHistogramModel = append(e.expHistogramModel, &expHistogramModel{
		metricName:        name,
		metricDescription: description,
		metricUnit:        unit,
		expHistogram:      expHistogram,
	})
}

func (e *ExpHistogramMetrics) InjectMetaData(metadata *MetricsMetaData) {
	e.metadata = metadata
}
