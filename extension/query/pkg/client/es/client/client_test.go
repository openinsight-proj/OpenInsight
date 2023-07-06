package client

import (
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/stretchr/testify/assert"
)

func TestParseBody(t *testing.T) {
	content, err := os.ReadFile("./testdata/span_search_results.json")
	if err != nil {
		log.Printf("read failed: %s", err)
	}
	res := &esapi.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(content)))}

	results, err := parseBody(res)
	if err != nil {
		log.Printf("parseBody failed: %s", err)
	}
	assert.Equal(t, 20, len(results.Hits.Hits))
}
