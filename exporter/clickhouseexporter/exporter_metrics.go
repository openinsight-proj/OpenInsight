package clickhouseexporter

import (
	"context"
	"database/sql"
	"fmt"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"strings"
	"time"
)

type metricsExporter struct {
	client *sql.DB

	logger *zap.Logger
	cfg    *Config
}

type metricsModel interface {
	doAdd(gauge pmetric.Gauge, name string, description string, unit string)
	doInjectMetaData(metaData *metricsMetaData)
	doInsert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error
}

type metricsMetaData struct {
	resAttr    map[string]string
	resUrl     string
	scopeUrl   string
	scopeInstr pcommon.InstrumentationScope
}

type gaugeModel struct {
	MetricName        string
	MetricDescription string
	MetricUnit        string
	gauge             pmetric.Gauge
}

type gaugeMetrics struct {
	gaugeModels []*gaugeModel
	metadata    *metricsMetaData
	insertSQL   string
}

func (g *gaugeMetrics) doInsert(ctx context.Context, tx *sql.Tx, logger *zap.Logger) error {
	start := time.Now()
	//do insert
	statement, err := tx.PrepareContext(ctx, g.insertSQL)
	if err != nil {
		return fmt.Errorf("PrepareContext:%w", err)
	}
	defer func() {
		_ = statement.Close()
	}()
	count := 0
	for _, model := range g.gaugeModels {
		for i := 0; i < model.gauge.DataPoints().Len(); i++ {
			dp := model.gauge.DataPoints().At(i)
			attrs, times, floatValues, intValues, traceIDs, spanIDs := convertExemplars(dp.Exemplars())
			//todo 区分 ValueAsDouble ValueAsInt
			_, err = statement.ExecContext(ctx,
				g.metadata.resAttr,
				g.metadata.resUrl,
				g.metadata.scopeInstr.Name(),
				g.metadata.scopeInstr.Version(),
				g.metadata.scopeInstr.Attributes(),
				g.metadata.scopeUrl,
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

func (g *gaugeMetrics) doAdd(pGauge pmetric.Gauge, name string, description string, unit string) {
	g.gaugeModels = append(g.gaugeModels, &gaugeModel{
		MetricName:        name,
		MetricDescription: description,
		MetricUnit:        unit,
		gauge:             pGauge,
	})
}

func (g *gaugeMetrics) doInjectMetaData(metaData *metricsMetaData) {
	g.metadata = metaData
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
			metaData := metricsMetaData{}
			metrics := md.ResourceMetrics().At(i)
			res := metrics.Resource()
			metaData.resAttr = attributesToMap(res.Attributes())
			metaData.resUrl = metrics.SchemaUrl()
			for j := 0; j < metrics.ScopeMetrics().Len(); j++ {
				rs := metrics.ScopeMetrics().At(j).Metrics()
				metaData.scopeUrl = metrics.ScopeMetrics().At(j).SchemaUrl()
				metaData.scopeInstr = metrics.ScopeMetrics().At(j).Scope()
				for k := 0; k < rs.Len(); k++ {
					r := rs.At(k)
					switch r.Type() {
					case pmetric.MetricTypeGauge:
						metricsMap[pmetric.MetricTypeGauge].doAdd(r.Gauge(), r.Name(), r.Description(), r.Unit())
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

		if err := doInsert(ctx, tx, metricsMap, e.logger); err != nil {
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

func createMetricsModel(cfg *Config) map[pmetric.MetricType]metricsModel {
	metricsMap := make(map[pmetric.MetricType]metricsModel)
	metricsMap[pmetric.MetricTypeGauge] = &gaugeMetrics{
		insertSQL: fmt.Sprintf(strings.ReplaceAll(insertGaugeTableSQL, "'", "`"), cfg.TracesTableName),
	}
	//todo others
	return metricsMap
}

func injectMetaData(metricsMap map[pmetric.MetricType]metricsModel, metaData *metricsMetaData) {
	for _, metrics := range metricsMap {
		metrics.doInjectMetaData(metaData)
	}
}

func doInsert(ctx context.Context, tx *sql.Tx, metricsMap map[pmetric.MetricType]metricsModel, logger *zap.Logger) error {
	for _, metrics := range metricsMap {
		if err := metrics.doInsert(ctx, tx, logger); err != nil {
			return err
		}
	}
	return nil
}

func convertExemplars(exemplars pmetric.ExemplarSlice) ([]map[string]string, []time.Time, []float64, []int64, []string, []string) {
	var (
		attrs       []map[string]string
		times       []time.Time
		floatValues []float64
		intValues   []int64
		traceIDs    []string
		spanIDs     []string
	)
	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)
		attrs = append(attrs, attributesToMap(exemplar.FilteredAttributes()))
		times = append(times, exemplar.Timestamp().AsTime())
		floatValues = append(floatValues, exemplar.DoubleValue())
		intValues = append(intValues, exemplar.IntValue())
		traceIDs = append(traceIDs, exemplar.TraceID().HexString())
		spanIDs = append(spanIDs, exemplar.SpanID().HexString())
	}
	return attrs, times, floatValues, intValues, traceIDs, spanIDs
}
