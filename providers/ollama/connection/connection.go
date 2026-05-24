// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"

	"github.com/ollama/ollama/api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	HostOption  = "host"
	TokenOption = "token"
)

type OllamaConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *api.Client
	host   string
}

func NewOllamaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OllamaConnection, error) {
	host := conf.Options[HostOption]
	if host == "" {
		host = os.Getenv("OLLAMA_HOST")
	}
	if host == "" {
		host = "http://localhost:11434"
	}

	token := conf.Options[TokenOption]
	if token == "" {
		token = os.Getenv("OLLAMA_API_TOKEN")
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	if conf.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	if token != "" {
		httpClient.Transport = &tokenTransport{
			token: token,
			base:  httpClient.Transport,
		}
	}

	client := api.NewClient(baseURL, httpClient)

	conn := &OllamaConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     client,
		host:       baseURL.String(),
	}

	return conn, nil
}

func (c *OllamaConnection) Name() string {
	return "ollama"
}

func (c *OllamaConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *OllamaConnection) Client() *api.Client {
	return c.client
}

func (c *OllamaConnection) Host() string {
	return c.host
}

type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
