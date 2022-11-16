package clickhouseexporter

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"
	"log"
	"strings"
	"testing"
	"time"
)

func simpleMetrics(count int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	for i := 0; i < count; i++ {
		// gauge
		m := sm.Metrics().AppendEmpty()
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetIntValue(int64(i))
		dp.Attributes().PutStr("gauge_label_1", "1")
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		exemplars := dp.Exemplars().AppendEmpty()
		exemplars.SetIntValue(54)
		exemplars.FilteredAttributes().PutStr("key", "value")
		exemplars.FilteredAttributes().PutStr("key2", "value2")

		// sum
		m = sm.Metrics().AppendEmpty()
		dp = m.SetEmptySum().DataPoints().AppendEmpty()
		dp.SetIntValue(int64(i))
		dp.Attributes().PutStr("sum_label_1", "1")
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		exemplars = dp.Exemplars().AppendEmpty()
		exemplars.SetIntValue(54)
		exemplars.FilteredAttributes().PutStr("key", "value")
		exemplars.FilteredAttributes().PutStr("key2", "value2")

		// histogram
		m = sm.Metrics().AppendEmpty()
		dpHisto := m.SetEmptyHistogram().DataPoints().AppendEmpty()
		dpHisto.SetCount(1)
		dpHisto.SetSum(1)
		dpHisto.Attributes().PutStr("key", "value")
		dpHisto.Attributes().PutStr("key", "value")
		dpHisto.ExplicitBounds().FromRaw([]float64{0, 0, 0, 0, 0})
		dpHisto.BucketCounts().FromRaw([]uint64{0, 0, 0, 1, 0})
		dpHisto.SetMin(0)
		dpHisto.SetMax(1)
		exemplars = dpHisto.Exemplars().AppendEmpty()
		exemplars.SetIntValue(54)
		exemplars.FilteredAttributes().PutStr("key", "value")
		exemplars.FilteredAttributes().PutStr("key2", "value2")

		// exp histogram
		m = sm.Metrics().AppendEmpty()
		dpExpHisto := m.SetEmptyExponentialHistogram().DataPoints().AppendEmpty()
		dpExpHisto.SetSum(1)
		dpExpHisto.SetMin(0)
		dpExpHisto.SetMax(1)
		dpExpHisto.SetZeroCount(0)
		dpExpHisto.SetCount(1)
		dpExpHisto.Attributes().PutStr("key", "value")
		dpExpHisto.Attributes().PutStr("key", "value")
		dpExpHisto.Negative().SetOffset(1)
		dpExpHisto.Negative().BucketCounts().FromRaw([]uint64{0, 0, 0, 1, 0})
		dpExpHisto.Positive().SetOffset(1)
		dpExpHisto.Positive().BucketCounts().FromRaw([]uint64{0, 0, 0, 1, 0})

		exemplars = dpHisto.Exemplars().AppendEmpty()
		exemplars.SetIntValue(54)
		exemplars.FilteredAttributes().PutStr("key", "value")
		exemplars.FilteredAttributes().PutStr("key2", "value2")

		// summary
		m = sm.Metrics().AppendEmpty()
		summary := m.SetEmptySummary().DataPoints().AppendEmpty()
		summary.Attributes().PutStr("key", "value")
		summary.Attributes().PutStr("key2", "value2")
		summary.SetCount(1)
		summary.SetSum(1)
		quantileValues := summary.QuantileValues().AppendEmpty()
		quantileValues.SetValue(1)
		quantileValues.SetQuantile(1)
	}
	return metrics
}

func mustPushMetricsData(t *testing.T, exporter *metricsExporter, md pmetric.Metrics) {
	err := exporter.pushMetricsData(context.TODO(), md)
	require.NoError(t, err)
}

func newTestMetricsExporter(t *testing.T, dsn string, fns ...func(*Config)) *metricsExporter {
	exporter, err := newMetricsExporter(zaptest.NewLogger(t), withTestExporterConfig(fns...)(dsn))
	require.NoError(t, err)

	t.Cleanup(func() { _ = exporter.Shutdown(context.TODO()) })
	return exporter
}

func TestExporter_pushMetricsData(t *testing.T) {
	t.Run("push sucess", func(t *testing.T) {
		var items int
		initClickhouseTestServer(t, func(query string, values []driver.Value) error {
			t.Logf("%d, values:%+v", items, values)
			if strings.HasPrefix(query, "INSERT") {
				items++
			}
			return nil
		})
		exporter := newTestMetricsExporter(t, defaultDSN)
		mustPushMetricsData(t, exporter, simpleMetrics(2))

		require.Equal(t, 5, items)
	})
}

// local dev test
func Test_newMetricsExporter(t *testing.T) {
	exporter := newTestMetricsExporter(t, defaultDSN)
	mustPushMetricsData(t, exporter, simpleMetrics(2))
}

// still in process
func Test_tran(t *testing.T) {
	exporter := newTestMetricsExporter(t, defaultDSN)
	db := exporter.client
	tx, err := db.Begin()
	if err != nil {
		fmt.Errorf("db.Begin: %w", err)
	}

	createSQL1 := `
CREATE TABLE IF NOT EXISTS foo (
    IntValue Int64,
    Exemplars Nested (
		Attributes Map(LowCardinality(String), String)
    ) CODEC(ZSTD(1))          
) ENGINE MergeTree()
PARTITION BY IntValue
ORDER BY IntValue
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;`

	insertSQL1 := "INSERT INTO foo (IntValue) VALUES (?)"

	createSQL2 := `
CREATE TABLE IF NOT EXISTS bar (
    IntValue Int64,
    Exemplars Nested (
		Attributes Map(LowCardinality(String), String)
    ) CODEC(ZSTD(1))          
) ENGINE MergeTree()
PARTITION BY IntValue
ORDER BY IntValue
SETTINGS index_granularity=8192, ttl_only_drop_parts = 1;`

	insertSQL2 := "INSERT INTO bar (IntValue) VALUES (?)"

	valueArgs := []interface{}{
		int64(14),
	}

	_, err = db.Exec(createSQL1)
	require.NoError(t, err)
	_, err = db.Exec(createSQL2)
	require.NoError(t, err)

	ctx := context.Background()
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	//use Prepare error:
	//table 1
	//dbStmt, err := db.Prepare(insertSQL1)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//_, err = tx.Stmt(dbStmt).Exec(valueArgs...)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//table 2
	//dbStmt, err = db.Prepare(insertSQL2)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//_, err = tx.Stmt(dbStmt).Exec(valueArgs...)
	//if err != nil {
	//	log.Fatal(err)
	//}

	_, err = tx.ExecContext(ctx, insertSQL1, valueArgs...)
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, insertSQL2, valueArgs...)
	if err != nil {
		log.Fatal(err)
	}

	err = tx.Rollback()
	if err != nil {
		log.Fatal(err)
	}

}
