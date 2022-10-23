package es

import (
	"encoding/json"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/query/pkg/client/es/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func mockSearchHits() *client.SearchHits {
	json1 := json.RawMessage(`{"@timestamp":"2022-09-23T09:51:57.610168000Z","Attributes.http.flavor":"1.1","Attributes.http.host":"0.0.0.0:7080","Attributes.http.method":"GET","Attributes.http.scheme":"http","Attributes.http.status_code":200,"Attributes.http.url":"http://0.0.0.0:7080/hello","EndTimestamp":"2022-09-23T09:51:57.882327833Z","Kind":"SPAN_KIND_CLIENT","Link":"[]","Name":"HTTP GET","ParentSpanId":"4c90353fd38bfc6d","Resource.host.name":"MacBook-Pro.local","Resource.process.command_args":["/private/var/folders/bw/y7g4n5ns17x2207szr9zx4qm0000gn/T/GoLand/___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_client"],"Resource.process.executable.name":"___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_client","Resource.process.executable.path":"/private/var/folders/bw/y7g4n5ns17x2207szr9zx4qm0000gn/T/GoLand/___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_client","Resource.process.owner":"tanjian","Resource.process.pid":90548,"Resource.process.runtime.description":"go version go1.18.3 darwin/amd64","Resource.process.runtime.name":"go","Resource.process.runtime.version":"go1.18.3","Resource.service.name":"demo-client","Resource.telemetry.sdk.language":"go","Resource.telemetry.sdk.name":"opentelemetry","Resource.telemetry.sdk.version":"1.9.0","SpanId":"7991e8d601df8e73","TraceId":"393b286a086c289d067bc30ddc6c0923","TraceStatus":0}`)
	json2 := json.RawMessage(`{"@timestamp":"2022-09-23T09:52:02.047114000Z","Attributes.baggage key:client":"cli","Attributes.baggage key:method":"repl","Attributes.http.flavor":"1.1","Attributes.http.host":"0.0.0.0:7080","Attributes.http.method":"GET","Attributes.http.scheme":"http","Attributes.http.server_name":"/hello","Attributes.http.status_code":200,"Attributes.http.target":"/hello","Attributes.http.user_agent":"Go-http-client/1.1","Attributes.http.wrote_bytes":11,"Attributes.net.host.ip":"0.0.0.0","Attributes.net.host.port":7080,"Attributes.net.peer.ip":"127.0.0.1","Attributes.net.peer.port":58874,"Attributes.net.transport":"ip_tcp","Attributes.server-attribute":"foo","EndTimestamp":"2022-09-23T09:52:02.859418813Z","Kind":"SPAN_KIND_SERVER","Link":"[]","Name":"/hello","ParentSpanId":"113e6afd0c933e12","Resource.host.name":"MacBook-Pro.local","Resource.process.command_args":["/private/var/folders/bw/y7g4n5ns17x2207szr9zx4qm0000gn/T/GoLand/___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_server"],"Resource.process.executable.name":"___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_server","Resource.process.executable.path":"/private/var/folders/bw/y7g4n5ns17x2207szr9zx4qm0000gn/T/GoLand/___go_build_github_com_open_telemetry_opentelemetry_collector_contrib_examples_demo_server","Resource.process.owner":"tanjian","Resource.process.pid":90497,"Resource.process.runtime.description":"go version go1.18.3 darwin/amd64","Resource.process.runtime.name":"go","Resource.process.runtime.version":"go1.18.3","Resource.service.name":"demo-server","Resource.telemetry.sdk.language":"go","Resource.telemetry.sdk.name":"opentelemetry","Resource.telemetry.sdk.version":"1.9.0","SpanId":"34fad113dc25cd07","TraceId":"9c38cfab65e7db46f16d2167d3ef9ee1","TraceStatus":0}`)
	searchHits := &client.SearchHits{
		Hits: []*client.SearchHit{
			{
				Index:  "otlp_spans",
				Type:   "_doc",
				Source: &json1,
			},
			{
				Index:  "otlp_spans",
				Type:   "_doc",
				Source: &json2,
			},
		},
	}
	return searchHits
}
func Test_DocumentsConvert(t *testing.T) {
	tracesData, err := DocumentsResourceSpansConvert(mockSearchHits())
	require.NoError(t, err)
	assert.Equal(t, 2, len(tracesData.ResourceSpans))
}
