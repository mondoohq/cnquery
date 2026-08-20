// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

const (
	HostOption  = "host"
	TokenOption = "token"
)

// probeTimeout bounds the unauthenticated probe. It is a single request against
// an instance we already talk to, so a short budget is enough and keeps a
// wedged server from stalling the query.
const probeTimeout = 10 * time.Second

type OllamaConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client *api.Client
	// baseURL is the parsed host, kept so the scheme and hostname can be
	// inspected without re-parsing and so probes can build their own requests.
	baseURL *url.URL
	// baseTransport carries the TLS settings but never the API token, so an
	// unauthenticated probe can reuse it without leaking credentials.
	baseTransport http.RoundTripper

	versionOnce sync.Once
	version     string
	versionErr  error
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

	var baseTransport http.RoundTripper
	if conf.Insecure {
		baseTransport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	httpClient := &http.Client{Transport: baseTransport}
	if token != "" {
		httpClient.Transport = &tokenTransport{
			token: token,
			base:  baseTransport,
		}
	}

	client := api.NewClient(baseURL, httpClient)

	conn := &OllamaConnection{
		Connection:    plugin.NewConnection(id, asset),
		Conf:          conf,
		asset:         asset,
		client:        client,
		baseURL:       baseURL,
		baseTransport: baseTransport,
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
	return c.baseURL.String()
}

// TLS reports whether the instance is reached over an encrypted connection.
func (c *OllamaConnection) TLS() bool {
	return strings.EqualFold(c.baseURL.Scheme, "https")
}

// IsLocal reports whether the configured host is a loopback address, meaning
// the API is only reachable from the machine running the server. Only the
// configured address is inspected; no name resolution is performed, so a
// hostname that happens to resolve to 127.0.0.1 reports false.
func (c *OllamaConnection) IsLocal() bool {
	hostname := c.baseURL.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(hostname, "localhost") ||
		strings.HasSuffix(strings.ToLower(hostname), ".localhost")
}

// Version returns the version reported by the instance. The result is fetched
// once and reused, so asset detection and the version field share one call.
func (c *OllamaConnection) Version(ctx context.Context) (string, error) {
	c.versionOnce.Do(func() {
		c.version, c.versionErr = c.client.Version(ctx)
	})
	return c.version, c.versionErr
}

// AnonymousStatus issues a read-only request with no credentials attached and
// returns the HTTP status code the instance answers with. It deliberately uses
// the model listing endpoint: it is a plain GET that changes nothing, so the
// probe cannot alter the instance it is auditing.
func (c *OllamaConnection) AnonymousStatus(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL.JoinPath("/api/tags").String(), nil)
	if err != nil {
		return 0, err
	}

	// A fresh client, so the token-bearing transport cannot be reached.
	client := &http.Client{Transport: c.baseTransport, Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to probe %s for anonymous access: %w", c.baseURL.Host, err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
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
