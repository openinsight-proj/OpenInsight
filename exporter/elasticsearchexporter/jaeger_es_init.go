// Copyright The OpenInsight Authors
// SPDX-License-Identifier: Apache-2.0

package elasticsearchexporter

import (
	"context"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"go.uber.org/zap"
	"io"
	"strconv"
	"strings"
)

type elasticsearchInit struct {
	client *esClientCurrent
	log    *zap.Logger
	esCase []initCase
	esILM  ILM
}

type ILM struct {
	ShardNum           int64  `mapstructure:"shards_num"`
	ReplicaNum         int64  `mapstructure:"replica_num"`
	RefreshInterval    string `mapstructure:"refresh_interval"`
	TranslogDurability string `mapstructure:"translog_durability"`
	// max_primary_shard_size: This is the maximum size of the primary shards in the index. As with max_size, replicas are ignored.
	MaxShardsSize string `mapstructure:"max_primary_shard_size"`
	// max_size: This is the total size of all primary shards in the index. Replicas are not counted toward the maximum index size.
	MaxSize string `mapstructure:"max_size"`
	MaxAge  string `mapstructure:"max_age"`
	TTL     string `mapstructure:"ttl"`
}

type initCase struct {
	templateName  string
	policyName    string
	initIndexName string

	templateStr  string
	policyStr    string
	initIndexStr string
}

var (
	ilmConfig ILM
)

func (e *elasticsearchInit) init() {
	//esConfig := i..Datasource.ES
	ilmConfig = e.esILM

	if ilmConfig.ShardNum <= 0 {
		ilmConfig.ShardNum = 1
	}

	if ilmConfig.ReplicaNum < 0 {
		ilmConfig.ReplicaNum = 0
	}

	if len(ilmConfig.RefreshInterval) == 0 {
		ilmConfig.RefreshInterval = "5s"
	}
	if len(ilmConfig.TranslogDurability) == 0 {
		ilmConfig.TranslogDurability = "async"
	}

	if len(ilmConfig.MaxShardsSize) == 0 {
		ilmConfig.MaxShardsSize = "10gb"
	}

	if len(ilmConfig.MaxAge) == 0 {
		ilmConfig.MaxAge = "7d"
	}

	if len(ilmConfig.MaxSize) == 0 {
		ilmConfig.MaxSize = "20gb"
	}

	if len(ilmConfig.TTL) == 0 {
		ilmConfig.TTL = "30d"
	}

	policy := "{\"policy\":{\"phases\":{\"hot\":{\"min_age\":\"0ms\",\"actions\":{\"forcemerge\":{\"max_num_segments\":1},\"rollover\":{\"max_primary_shard_size\":\"" + ilmConfig.MaxShardsSize + "\", \"max_size\":\"" + ilmConfig.MaxSize + "\" , \"max_age\" : \"" + ilmConfig.MaxAge + "\"}}},\"delete\":{\"min_age\":\"" + ilmConfig.TTL + "\",\"actions\":{\"delete\":{\"delete_searchable_snapshot\":true}}}}}}"
	e.esCase = []initCase{
		{
			// jaeger service case
			templateName:  "jaeger-service",
			policyName:    "jaeger-ilm-policy",
			initIndexName: "jaeger-service-000001",
			templateStr:   "{\"order\":0,\"index_patterns\":[\"*jaeger-service-*\"],\"settings\":{\"index\":{\"lifecycle\":{\"name\":\"jaeger-ilm-policy\",\"rollover_alias\":\"jaeger-service-write\"},\"mapping\":{\"nested_fields\":{\"limit\":\"50\"}},\"requests\":{\"cache\":{\"enable\":\"true\"}},\"number_of_shards\":\"1\",\"number_of_replicas\":\"0\"}},\"mappings\":{\"dynamic_templates\":[{\"span_tags_map\":{\"path_match\":\"tag.*\",\"mapping\":{\"ignore_above\":256,\"type\":\"keyword\"}}},{\"process_tags_map\":{\"path_match\":\"process.tag.*\",\"mapping\":{\"ignore_above\":256,\"type\":\"keyword\"}}}],\"properties\":{\"operationName\":{\"ignore_above\":256,\"type\":\"keyword\"},\"serviceName\":{\"ignore_above\":256,\"type\":\"keyword\"}}},\"aliases\":{\"jaeger-service-read\":{}}}",
			policyStr:     policy,
			initIndexStr:  "{\"aliases\": {\"jaeger-service-write\":{\"is_write_index\": true }}}",
		},
		{
			// jaeger span case
			templateName:  "jaeger-span",
			policyName:    "",
			initIndexName: "jaeger-span-000001",
			templateStr:   "{\"order\":0,\"index_patterns\":[\"*jaeger-span-*\"],\"settings\":{\"index\":{\"lifecycle\":{\"name\":\"jaeger-ilm-policy\",\"rollover_alias\":\"jaeger-span-write\"},\"mapping\":{\"nested_fields\":{\"limit\":\"50\"}},\"requests\":{\"cache\":{\"enable\":\"true\"}},\"number_of_shards\":\"" + strconv.Itoa(int(ilmConfig.ShardNum)) + "\",\"number_of_replicas\":\"" + strconv.Itoa(int(ilmConfig.ReplicaNum)) + "\"}},\"mappings\":{\"dynamic_templates\":[{\"span_tags_map\":{\"path_match\":\"tag.*\",\"mapping\":{\"ignore_above\":256,\"type\":\"keyword\"}}},{\"process_tags_map\":{\"path_match\":\"process.tag.*\",\"mapping\":{\"ignore_above\":256,\"type\":\"keyword\"}}}],\"properties\":{\"traceID\":{\"ignore_above\":256,\"type\":\"keyword\"},\"process\":{\"properties\":{\"tag\":{\"type\":\"object\"},\"serviceName\":{\"ignore_above\":256,\"type\":\"keyword\"},\"tags\":{\"dynamic\":false,\"type\":\"nested\",\"properties\":{\"tagType\":{\"ignore_above\":256,\"type\":\"keyword\"},\"value\":{\"ignore_above\":256,\"type\":\"keyword\"},\"key\":{\"ignore_above\":256,\"type\":\"keyword\"}}}}},\"startTimeMillis\":{\"format\":\"epoch_millis\",\"type\":\"date\"},\"references\":{\"dynamic\":false,\"type\":\"nested\",\"properties\":{\"traceID\":{\"ignore_above\":256,\"type\":\"keyword\"},\"spanID\":{\"ignore_above\":256,\"type\":\"keyword\"},\"refType\":{\"ignore_above\":256,\"type\":\"keyword\"}}},\"flags\":{\"type\":\"integer\"},\"operationName\":{\"ignore_above\":256,\"type\":\"keyword\"},\"parentSpanID\":{\"ignore_above\":256,\"type\":\"keyword\"},\"tags\":{\"dynamic\":false,\"type\":\"nested\",\"properties\":{\"tagType\":{\"ignore_above\":256,\"type\":\"keyword\"},\"value\":{\"ignore_above\":256,\"type\":\"keyword\"},\"key\":{\"ignore_above\":256,\"type\":\"keyword\"}}},\"spanID\":{\"ignore_above\":256,\"type\":\"keyword\"},\"duration\":{\"type\":\"long\"},\"startTime\":{\"type\":\"long\"},\"tag\":{\"type\":\"object\"},\"logs\":{\"dynamic\":false,\"type\":\"nested\",\"properties\":{\"fields\":{\"dynamic\":false,\"type\":\"nested\",\"properties\":{\"tagType\":{\"ignore_above\":256,\"type\":\"keyword\"},\"value\":{\"ignore_above\":256,\"type\":\"keyword\"},\"key\":{\"ignore_above\":256,\"type\":\"keyword\"}}},\"timestamp\":{\"type\":\"long\"}}}}},\"aliases\":{\"jaeger-span-read\":{}}}",
			policyStr:     "",
			initIndexStr:  "{\"aliases\": {\"jaeger-span-write\":{\"is_write_index\": true }}}",
		},
		{
			// jaeger dependencies case
			templateName:  "jaeger-dependencies",
			policyName:    "",
			initIndexName: "jaeger-dependencies-000001",
			templateStr:   "{\"order\":0,\"index_patterns\":[\"*jaeger-dependencies-*\"],\"settings\":{\"index\":{\"lifecycle\":{\"name\":\"jaeger-ilm-policy\",\"rollover_alias\":\"jaeger-dependencies-write\"},\"mapping\":{\"nested_fields\":{\"limit\":\"50\"}},\"requests\":{\"cache\":{\"enable\":\"true\"}},\"number_of_shards\":\"" + strconv.Itoa(int(ilmConfig.ShardNum)) + "\",\"number_of_replicas\":\"" + strconv.Itoa(int(ilmConfig.ReplicaNum)) + "\"}},\"mappings\":{},\"aliases\":{\"jaeger-dependencies-read\":{}}}",
			policyStr:     "",
			initIndexStr:  "{\"aliases\": {\"jaeger-dependencies-write\":{\"is_write_index\": true }}}",
		},
	}

	e.checkAndInitElasticsearch()
}

func (e elasticsearchInit) checkAndInitElasticsearch() {
	for _, cs := range e.esCase {
		e.log.Info("[Init Elasticsearch] init", zap.String("template name", cs.templateName))

		var exists bool
		// 1. check and update _index_template
		if cs.templateName != "" {
			exists, _ = e.existsTemplate(cs.templateName)
			if !exists {
				ok, err := e.createTemplate(cs.templateName, cs.templateStr)
				if ok {
					e.log.Info("[SUCCESS] create _index_template success.", zap.String("template name", cs.templateName))
				}
				if err != nil {
					e.log.Info("[FAIL] create _index_template  failed.", zap.String("template name", cs.templateName))
				}
			}
		}

		// 2. check and update ILM policy
		if cs.policyName != "" {
			exists, _ = e.existsILMPolicy(cs.policyName)
			if !exists {
				ok, err := e.createILMPolicy(cs.policyName, cs.policyStr)
				if ok {
					e.log.Info("[SUCCESS] create ilm policy success.", zap.String("policy name", cs.templateName))
				}
				if err != nil {
					e.log.Info("[FAIL] create ilm policy: %s failed.", zap.String("policy name", cs.templateName))
				}
			}

		}

		// 3. create increment index(*-00001) as rollover starter.
		if cs.templateName != "" {
			exists, _ = e.existsInitIndices(cs.templateName)
			if !exists {
				ok, err := e.createFirstIndex(cs.initIndexName, cs.initIndexStr)
				if ok {
					e.log.Info("[SUCCESS] create increments index success.", zap.String("index name", cs.initIndexName))
				}
				if err != nil {
					e.log.Info("[FAIL] create increments index failed.", zap.String("index name", cs.initIndexName))
				}
			}
		}
	}
}

func (e elasticsearchInit) existsTemplate(name string) (bool, error) {
	var resp *esapi.Response
	if strings.Contains(name, "jaeger") {
		templateRequest := &esapi.IndicesExistsTemplateRequest{
			Name: []string{name},
		}
		resp, _ = templateRequest.Do(context.Background(), e.client)
	} else {
		indexTemplateRequest := &esapi.IndicesGetIndexTemplateRequest{
			Name: name,
		}
		resp, _ = indexTemplateRequest.Do(context.Background(), e.client)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return false, fmt.Errorf("template: %s not found.", name)
	}

	if resp.StatusCode == 200 {
		var resultStr string
		if b, err := io.ReadAll(resp.Body); err == nil {
			resultStr = string(b)
		}
		e.log.Info("template exists, skip init...", zap.String("result", resultStr))
	}
	return true, nil
}

func (e elasticsearchInit) existsILMPolicy(policy string) (bool, error) {
	ilmRequest := &esapi.ILMGetLifecycleRequest{
		Policy: policy,
	}
	resp, _ := ilmRequest.Do(context.Background(), e.client)
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return false, fmt.Errorf("_ilm policy: %s not found.", policy)
	}

	if resp.StatusCode == 200 {
		var resultStr string
		if b, err := io.ReadAll(resp.Body); err == nil {
			resultStr = string(b)
		}
		e.log.Info("_ilm policy exists: %s, skip init...", zap.String("result", resultStr))
	}
	return true, nil
}

func (e elasticsearchInit) existsInitIndices(templateName string) (bool, error) {
	// get indices through `templatename-*` match indices
	indicesRequest := &esapi.IndicesGetRequest{
		Index: []string{templateName + "-*"},
	}
	resp, _ := indicesRequest.Do(context.Background(), e.client)
	defer resp.Body.Close()
	var resultStr string
	if b, err := io.ReadAll(resp.Body); err == nil {
		resultStr = string(b)
	}
	if resultStr != "{}" {
		e.log.Info("indices exists: %s, skip init...", zap.String("result", resultStr))
		return true, nil
	}
	return false, nil
}

func (e elasticsearchInit) createTemplate(name string, template string) (bool, error) {

	var resp *esapi.Response
	if strings.Contains(name, "jaeger") {
		templateRequest := &esapi.IndicesPutTemplateRequest{
			Name: name,
			Body: strings.NewReader(template),
		}
		resp, _ = templateRequest.Do(context.Background(), e.client)
	} else {
		// _index_template
		indexTemplateRequest := &esapi.IndicesPutIndexTemplateRequest{
			Name: name,
			Body: strings.NewReader(template),
		}
		resp, _ = indexTemplateRequest.Do(context.Background(), e.client)
	}

	defer resp.Body.Close()
	var resultStr string
	if b, err := io.ReadAll(resp.Body); err == nil {
		resultStr = string(b)
	}

	if resultStr == "{\"acknowledged\":true}" {
		return true, nil
	}

	return false, fmt.Errorf("create _index_template failed:%s", resultStr)
}

func (e elasticsearchInit) createILMPolicy(name string, policy string) (bool, error) {
	ilmPolicyRequest := &esapi.ILMPutLifecycleRequest{
		Policy: name,
		Body:   strings.NewReader(policy),
	}
	resp, _ := ilmPolicyRequest.Do(context.Background(), e.client)

	defer resp.Body.Close()
	var resultStr string
	if b, err := io.ReadAll(resp.Body); err == nil {
		resultStr = string(b)
	}

	if resultStr == "{\"acknowledged\":true}" {
		return true, nil
	}

	return false, fmt.Errorf("create ilm policy failed:%s", resultStr)
}

func (e elasticsearchInit) createFirstIndex(name string, indexStr string) (bool, error) {
	indicesRequest := &esapi.IndicesCreateRequest{
		Index: name,
		Body:  strings.NewReader(indexStr),
	}
	resp, _ := indicesRequest.Do(context.Background(), e.client)
	defer resp.Body.Close()

	var resultStr string
	if b, err := io.ReadAll(resp.Body); err == nil {
		resultStr = string(b)
	}

	if strings.Contains(resultStr, "{\"acknowledged\":true") {
		e.log.Info("create increments index.", zap.String("result", resultStr))
		return true, nil
	}

	return false, fmt.Errorf("create increments index failed:%s", resultStr)
}
