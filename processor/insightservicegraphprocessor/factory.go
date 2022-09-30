// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package insightservicegraphprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/insightservicegraphprocessor"

import (
	"context"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/insightservicegraphprocessor/internal/store"
	"time"

	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr   = "insight_service_graph"
	stability = component.StabilityLevelBeta
)

// NewFactory creates a factory for the servicegraph processor.
func NewFactory() component.ProcessorFactory {
	// TODO: Handle this err
	_ = view.Register(serviceGraphProcessorViews()...)

	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesProcessor(createTracesProcessor, stability),
	)
}

func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewComponentID(typeStr)),
		Store: store.Config{
			TTL:      2 * time.Second,
			MaxItems: 1000,
		},
	}
}

func createTracesProcessor(_ context.Context, params component.ProcessorCreateSettings, cfg config.Processor, nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	return newProcessor(params.Logger, cfg, nextConsumer)
}
