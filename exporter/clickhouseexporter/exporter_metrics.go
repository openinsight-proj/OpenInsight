package clickhouseexporter

import (
	"context"
	"database/sql"
	"fmt"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type metricsExporter struct {
	client *sql.DB

	logger *zap.Logger
	cfg    *Config
}

func newMetricsExporter(logger *zap.Logger, cfg *Config) (*metricsExporter, error) {
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
			metaData.ResUrl = metrics.SchemaUrl()
			for j := 0; j < metrics.ScopeMetrics().Len(); j++ {
				rs := metrics.ScopeMetrics().At(j).Metrics()
				metaData.ScopeUrl = metrics.ScopeMetrics().At(j).SchemaUrl()
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

		// batch https://clickhouse.com/docs/zh/introduction/performance/
		if err := internal.InsertMetrics(ctx, tx, metricsMap, e.logger); err != nil {
			//todo retry
			return fmt.Errorf("ExecContext:%w", err)
		}

		return nil
	})
	return err
}
