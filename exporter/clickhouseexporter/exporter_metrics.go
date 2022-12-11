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

package clickhouseexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter/internal"
)

type metricsExporter struct {
	client *sql.DB

	logger *zap.Logger
	cfg    *Config
}

func newMetricsExporter(logger *zap.Logger, cfg *Config) (*metricsExporter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := createDatabase(cfg); err != nil {
		return nil, err
	}
	client, err := newClickhouseClient(cfg)
	if err != nil {
		return nil, err
	}

	if err = internal.CreateMetricsTable(cfg.MetricsTableName, cfg.TTLDays, client); err != nil {
		return nil, err
	}

	return &metricsExporter{
		client: client,
		logger: logger,
		cfg:    cfg,
	}, nil
}

// Shutdown will shutdown the exporter.
func (e *metricsExporter) Shutdown(ctx context.Context) error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

func (e *metricsExporter) pushMetricsData(ctx context.Context, md pmetric.Metrics) error {
	err := doWithTx(ctx, e.client, func(tx *sql.Tx) error {
		metricsMap := internal.CreateMetricsModel(e.cfg.MetricsTableName)
		for i := 0; i < md.ResourceMetrics().Len(); i++ {
			metaData := internal.MetricsMetaData{}
			metrics := md.ResourceMetrics().At(i)
			res := metrics.Resource()
			metaData.ResAttr = attributesToMap(res.Attributes())
			metaData.ResURL = metrics.SchemaUrl()
			for j := 0; j < metrics.ScopeMetrics().Len(); j++ {
				rs := metrics.ScopeMetrics().At(j).Metrics()
				metaData.ScopeURL = metrics.ScopeMetrics().At(j).SchemaUrl()
				metaData.ScopeInstr = metrics.ScopeMetrics().At(j).Scope()
				for k := 0; k < rs.Len(); k++ {
					r := rs.At(k)
					switch r.Type() {
					case pmetric.MetricTypeGauge:
						metricsMap[pmetric.MetricTypeGauge].Add(r.Gauge(), r.Name(), r.Description(), r.Unit())
					case pmetric.MetricTypeSum:
						metricsMap[pmetric.MetricTypeSum].Add(r.Sum(), r.Name(), r.Description(), r.Unit())
					case pmetric.MetricTypeHistogram:
						metricsMap[pmetric.MetricTypeHistogram].Add(r.Histogram(), r.Name(), r.Description(), r.Unit())
					case pmetric.MetricTypeExponentialHistogram:
						metricsMap[pmetric.MetricTypeExponentialHistogram].Add(r.ExponentialHistogram(), r.Name(), r.Description(), r.Unit())
					case pmetric.MetricTypeSummary:
						metricsMap[pmetric.MetricTypeSummary].Add(r.Summary(), r.Name(), r.Description(), r.Unit())
					default:
						e.logger.Error("unsupported metrics type")
					}
				}
				internal.InjectMetaData(metricsMap, &metaData)
			}
		}

		// batch insert https://clickhouse.com/docs/en/about-us/performance/#performance-when-inserting-data
		if err := internal.InsertMetrics(ctx, tx, metricsMap, e.logger); err != nil {
			// TODO retry
			return fmt.Errorf("ExecContext:%w", err)
		}

		return nil
	})
	return err
}
