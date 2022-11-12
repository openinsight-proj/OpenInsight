package clickhouseexporter

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter/internal"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"strings"
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

	if err = createMetricsTable(cfg, client); err != nil {
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
		metricsMap := createMetricsModel(e.cfg)
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
					case pmetric.MetricTypeHistogram:
					case pmetric.MetricTypeExponentialHistogram:
					case pmetric.MetricTypeSummary:
					default:
						e.logger.Error("unsupported metrics type")
					}
				}
				//inject metadata
				injectMetaData(metricsMap, &metaData)
			}
		}

		if err := InsertMetrics(ctx, tx, metricsMap, e.logger); err != nil {
			return fmt.Errorf("ExecContext:%w", err)
		}
		//todo insert other metrics

		return nil
	})
	return err
}

const (
	// language=ClickHouse SQL
	createGaugeTableSQL = `
CREATE TABLE IF NOT EXISTS %s (
    ResourceAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ResourceSchemaUrl String CODEC(ZSTD(1)),
    ScopeName String CODEC(ZSTD(1)),
    ScopeVersion String CODEC(ZSTD(1)),
    ScopeAttributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    ScopeDroppedAttributesCount UInt32 CODEC(ZSTD(1)),
    ScopeSchemaUrl String CODEC(ZSTD(1)),
    MetricName String CODEC(ZSTD(1)),
    MetricDescription String CODEC(ZSTD(1)),
    MetricUnit String CODEC(ZSTD(1)),
    Attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    StartTimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    TimeUnix DateTime64(9) CODEC(Delta, ZSTD(1)),
    ValueAsDouble Float64 CODEC(ZSTD(1)),
    ValueAsInt UInt32 CODEC(ZSTD(1)),
    Flags UInt32  CODEC(ZSTD(1)),
    Exemplars Nested (
    filteredAttributes Map(LowCardinality(String), String),
    timeUnix DateTime64(9),
    valueAsDouble Float64,
    valueAsInt UInt32,
    spanId String,
    traceId) String
    ) CODEC(ZSTD(1))
) ENGINE MergeTree()
%s
PARTITION BY toDate(TimeUnix)
ORDER BY (toUnixTimestamp64Nano(TimeUnix))
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;
`
	// language=ClickHouse SQL
	insertGaugeTableSQL = `
INSERT INTO %s (
    ResourceAttributes,
    ResourceSchemaUrl,
    ScopeName,
    ScopeVersion,
    ScopeAttributes,
    ScopeSchemaUrl,
    MetricName,
    MetricDescription,
    MetricUnit,
    Attributes,
    TimeUnix,
    ValueAsDouble,
    ValueAsInt,
    Flags,
    Exemplars.filteredAttributes,
    Exemplars.timeUnix,
    Exemplars.valueAsDouble,
    Exemplars.valueAsInt,
    Exemplars.spanId,
    Exemplars.traceId) VALUES(
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
    )`
)

func createMetricsTable(cfg *Config, db *sql.DB) error {
	var ttlExpr string
	if cfg.TTLDays > 0 {
		ttlExpr = fmt.Sprintf(`TTL toDateTime(Timestamp) + toIntervalDay(%d)`, cfg.TTLDays)
	}
	if _, err := db.Exec(fmt.Sprintf(createGaugeTableSQL, cfg.MetricsTableName, ttlExpr)); err != nil {
		return fmt.Errorf("exec create traces table sql: %w", err)
	}
	return nil
}

func createMetricsModel(cfg *Config) map[pmetric.MetricType]internal.MetricsModel {
	metricsMap := make(map[pmetric.MetricType]internal.MetricsModel)
	metricsMap[pmetric.MetricTypeGauge] = &internal.GaugeMetrics{
		InsertSQL: fmt.Sprintf(strings.ReplaceAll(insertGaugeTableSQL, "'", "`"), cfg.TracesTableName),
	}
	//todo others
	return metricsMap
}

func injectMetaData(metricsMap map[pmetric.MetricType]internal.MetricsModel, metaData *internal.MetricsMetaData) {
	for _, metrics := range metricsMap {
		metrics.InjectMetaData(metaData)
	}
}

func InsertMetrics(ctx context.Context, tx *sql.Tx, metricsMap map[pmetric.MetricType]internal.MetricsModel, logger *zap.Logger) error {
	for _, metrics := range metricsMap {
		if err := metrics.Insert(ctx, tx, logger); err != nil {
			return err
		}
	}
	return nil
}
