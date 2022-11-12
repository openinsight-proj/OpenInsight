// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouseexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "clickhouse"
	// The stability level of the exporter.
	stability = component.StabilityLevelAlpha
)

// NewFactory creates a factory for Elastic exporter.
func NewFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithLogsExporter(createLogsExporter, stability),
		component.WithTracesExporter(createTracesExporter, stability),
		component.WithMetricsExporter(createMetricExporter, stability),
	)
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		TimeoutSettings:  exporterhelper.NewDefaultTimeoutSettings(),
		QueueSettings:    QueueSettings{QueueSize: exporterhelper.NewDefaultQueueSettings().QueueSize},
		RetrySettings:    exporterhelper.NewDefaultRetrySettings(),
		LogsTableName:    "otel_logs",
		TracesTableName:  "otel_traces",
		MetricsTableName: "otel_metrics",
		TTLDays:          7,
	}
}

// createLogsExporter creates a new exporter for logs.
// Logs are directly insert into clickhouse.
func createLogsExporter(
	ctx context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.LogsExporter, error) {
	c := cfg.(*Config)
	exporter, err := newLogsExporter(set.Logger, c)
	if err != nil {
		return nil, fmt.Errorf("cannot configure clickhouse logs exporter: %w", err)
	}

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		exporter.pushLogsData,
		exporterhelper.WithShutdown(exporter.Shutdown),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithQueue(c.enforcedQueueSettings()),
		exporterhelper.WithRetry(c.RetrySettings),
	)
}

// createTracesExporter creates a new exporter for traces.
// Traces are directly insert into clickhouse.
func createTracesExporter(
	ctx context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.TracesExporter, error) {
	c := cfg.(*Config)
	exporter, err := newTracesExporter(set.Logger, c)
	if err != nil {
		return nil, fmt.Errorf("cannot configure clickhouse traces exporter: %w", err)
	}

	return exporterhelper.NewTracesExporter(
		ctx,
		set,
		cfg,
		exporter.pushTraceData,
		exporterhelper.WithShutdown(exporter.Shutdown),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithQueue(c.enforcedQueueSettings()),
		exporterhelper.WithRetry(c.RetrySettings),
	)
}

func createMetricExporter(
	ctx context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	c := cfg.(*Config)
	exporter, err := newMetricsExporter(set.Logger, c)
	if err != nil {
		return nil, fmt.Errorf("cannot configure clickhouse metrics exporter: %w", err)
	}

	return exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		exporter.pushMetricsData,
		exporterhelper.WithShutdown(exporter.Shutdown),
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithQueue(c.enforcedQueueSettings()),
		exporterhelper.WithRetry(c.RetrySettings),
	)
}
