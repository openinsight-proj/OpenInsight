package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/aquasecurity/esquery"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Elastic struct {
	Client *elasticsearch.Client
}

func New(address []string, username, password string) (*Elastic, error) {
	var client *elasticsearch.Client
	var err error

	client, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: address,
		Username:  username,
		Password:  password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})

	return &Elastic{Client: client}, err
}

func (e *Elastic) Info() (*esapi.Response, error) {
	return e.Client.Info()
}

func parseError(response *esapi.Response) error {
	var e Error
	if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
		return err
	} else {
		// Print the response status and error information.
		if len(e.Details.RootCause) != 0 {
			return fmt.Errorf("type: %v, reason: %v", e.Details.Type, e.Details.RootCause[0].Reason)
		} else {
			return fmt.Errorf("type: %v, reason: %v", e.Details.Type, e.Details.Reason)
		}
	}
}

func (e *Elastic) DoSearch(ctx context.Context, index string, qsl *esquery.SearchRequest) (*SearchResult, error) {
	res, err := qsl.Run(e.Client, e.Client.Search.WithContext(ctx), e.Client.Search.WithIndex(index))
	if err != nil {
		zap.S().Error("Failed searching for stuff: %s", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.IsError() {
		return nil, parseError(res)
	}

	return parseBody(res)
}

func parseBody(response *esapi.Response) (*SearchResult, error) {
	// Return search results
	ret := new(SearchResult)
	ioBytes, err := io.ReadAll(response.Body)
	if err != nil {
		zap.S().Error("Failed read search result: %s", err)
	}
	if err := json.Unmarshal(ioBytes, &ret); err != nil {
		ret.Header = response.Header
		return ret, err
	}
	return ret, nil
}
