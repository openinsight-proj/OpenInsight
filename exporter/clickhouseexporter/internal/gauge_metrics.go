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

const gaugePlaceholders = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

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
	var valuePlaceholders []string
	var valueArgs []interface{}

	for _, model := range g.gaugeModels {
		for i := 0; i < model.gauge.DataPoints().Len(); i++ {
			dp := model.gauge.DataPoints().At(i)
			valuePlaceholders = append(valuePlaceholders, gaugePlaceholders)

			valueArgs = append(valueArgs, g.metadata.ResAttr)
			valueArgs = append(valueArgs, g.metadata.ResURL)
			valueArgs = append(valueArgs, g.metadata.ScopeInstr.Name())
			valueArgs = append(valueArgs, g.metadata.ScopeInstr.Version())
			valueArgs = append(valueArgs, attributesToMap(g.metadata.ScopeInstr.Attributes()))
			valueArgs = append(valueArgs, g.metadata.ScopeInstr.DroppedAttributesCount())
			valueArgs = append(valueArgs, g.metadata.ScopeURL)
			valueArgs = append(valueArgs, g.metadata.ServiceName)
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
		}
	}

	if len(valuePlaceholders) == 0 {
		return nil
	}

	start := time.Now()
	_, err := tx.ExecContext(ctx, fmt.Sprintf("%s %s", g.InsertSQL, strings.Join(valuePlaceholders, ",")), valueArgs...)
	if err != nil {
		return fmt.Errorf("insert gauge metrics fail:%w", err)
	}
	duration := time.Since(start)

	// TODO latency metrics
	logger.Info("insert gauge metrics", zap.Int("records", len(valuePlaceholders)),
		zap.String("cost", duration.String()))
	return nil
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

func (g *GaugeMetrics) InjectMetaData(metaData *MetricsMetaData) {
	g.metadata = metaData
}
