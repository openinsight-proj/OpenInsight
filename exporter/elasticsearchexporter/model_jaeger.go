// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package elasticsearchexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jaegertracing/jaeger/model"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter/internal/objmodel"
	semconv "go.opentelemetry.io/collector/semconv/v1.9.0"
	"hash/fnv"
	"time"

	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/jaegertracing/jaeger/plugin/storage/es/spanstore/dbmodel"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"

	otlp2jaeger "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
)

var (
	serviceCacheTTLDefault = 12 * time.Hour
)

type encodeJaegerModel struct {
	serviceCache cache.Cache
}

func NewEncodeJaegerModel() *encodeJaegerModel {
	return &encodeJaegerModel{serviceCache: cache.NewLRUWithOptions(
		100000,
		&cache.Options{
			TTL: serviceCacheTTLDefault,
		}),
	}
}

func (m *encodeJaegerModel) encodeSpan(resource pcommon.Resource, scope pcommon.InstrumentationScope, span ptrace.Span) ([]byte, error) {
	// TODO: make this domain converter configurable, refs: https://www.jaegertracing.io/docs/1.46/cli/#jaeger-all-in-one-elasticsearch
	fromDomain := dbmodel.NewFromDomain(false, []string{}, "@")
	td := ptrace.NewTraces()
	resourceSpan := td.ResourceSpans().AppendEmpty()
	resource.CopyTo(resourceSpan.Resource())
	ss := td.ResourceSpans().At(0).ScopeSpans().AppendEmpty()
	scope.CopyTo(ss.Scope())
	span.CopyTo(ss.Spans().AppendEmpty())

	singleBatch, err := otlp2jaeger.ProtoFromTraces(td)
	if err != nil {
		return nil, fmt.Errorf("otlp to jaeger error")
	}

	jSpan := singleBatch[0].GetSpans()[0]
	if singleBatch[0].Process != nil {
		jSpan.Process = singleBatch[0].Process
	} else {
		//TODO(jian): add process
		jSpan.Process = &model.Process{}
	}
	convertedSpan := fromDomain.FromDomainEmbedProcess(jSpan)
	return json.Marshal(convertedSpan)
}

func (m *encodeJaegerModel) encodeLog(_ pcommon.Resource, _ plog.LogRecord) ([]byte, error) {
	// do nothing
	return nil, nil
}

// encodeServiceNameOperation: will return service name and operation with _id when get metadata from attributes
func (m *encodeJaegerModel) encodeServiceNameOperation(resource pcommon.Resource, span ptrace.Span) (string, []byte, error) {
	serviceName, ok := findAttributeValue(semconv.AttributeServiceName, resource.Attributes())
	if !ok {
		return "", nil, fmt.Errorf("serviename not found")
	}

	// TODO(jian): we can add more attributes into service model, such as k8s metadata.
	service := dbmodel.Service{
		ServiceName:   serviceName,
		OperationName: span.Name(),
	}

	cacheKey := hashCode(service)
	if !keyInCache(cacheKey, m.serviceCache) {
		writeCache(cacheKey, m.serviceCache)
	}

	var document objmodel.Document
	document.AddString("serviceName", service.ServiceName)
	document.AddString("operationName", service.OperationName)
	var buf bytes.Buffer
	err := document.Serialize(&buf, true)
	return cacheKey, buf.Bytes(), err
}

func hashCode(s dbmodel.Service) string {
	h := fnv.New64a()
	h.Write([]byte(s.ServiceName))
	h.Write([]byte(s.OperationName))
	return fmt.Sprintf("%x", h.Sum64())
}

func keyInCache(key string, c cache.Cache) bool {
	return c.Get(key) != nil
}

func writeCache(key string, c cache.Cache) {
	c.Put(key, key)
}

func findAttributeValue(key string, attributes ...pcommon.Map) (string, bool) {
	for _, attr := range attributes {
		if v, ok := attr.Get(key); ok {
			return v.AsString(), true
		}
	}
	return "", false
}
