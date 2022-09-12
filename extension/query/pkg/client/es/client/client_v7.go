package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"net/http"
)

type Elastic struct {
	Client *elasticsearch.Client
}

func New(address string, username, password, index string) (*Elastic, error) {
	var client *elasticsearch.Client
	var err error

	client, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{address},
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
