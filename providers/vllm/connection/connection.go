// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	OptionAPIKey  = "api-key"
	OptionBaseURL = "base-url"
	OptionTimeout = "timeout"

	DefaultTimeoutSeconds = 10

	endpointProbeWorkers = 8
	maxProbeBody         = 64 << 10
	maxVersionBody       = 1 << 20

	corsProbeOrigin  = "https://mondoo.example"
	corsProbeHeaders = "authorization,content-type"
)

type EndpointSpec struct {
	Method   string
	Path     string
	Category string
	Body     string
}

type EndpointObservation struct {
	Spec                    EndpointSpec
	AnonymousStatusCode     *int
	AuthenticatedStatusCode *int
	AnonymousError          string
	AuthenticatedError      string
}

type VllmConnection struct {
	plugin.Connection
	Conf    *inventory.Config
	asset   *inventory.Asset
	client  *http.Client
	baseURL string
	apiKey  string

	endpointsOnce sync.Once
	endpoints     []EndpointObservation
	endpointsErr  error

	corsOnce sync.Once
	cors     CORSObservation
	corsErr  error

	versionOnce sync.Once
	version     string
	versionErr  error
}

type CORSObservation struct {
	Configured      *bool
	AllowsAnyOrigin *bool
	StatusCode      *int
	AllowOrigin     string
}

func NewVllmConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*VllmConnection, error) {
	baseURL, err := baseURLFromConfig(conf)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(DefaultTimeoutSeconds) * time.Second
	if conf.Options != nil {
		if raw := strings.TrimSpace(conf.Options[OptionTimeout]); raw != "" {
			seconds, err := strconv.Atoi(raw)
			if err != nil || seconds <= 0 {
				return nil, fmt.Errorf("vllm: invalid timeout %q", raw)
			}
			timeout = time.Duration(seconds) * time.Second
		}
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if conf.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user-controlled flag for lab/test environments
	}

	conn := &VllmConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: baseURL,
		apiKey:  apiKeyFromConfig(conf),
	}

	return conn, nil
}

func (c *VllmConnection) Name() string {
	return "vllm"
}

func (c *VllmConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *VllmConnection) Close() {
	if c.client != nil {
		c.client.CloseIdleConnections()
	}
}

func (c *VllmConnection) BaseURL() string {
	return c.baseURL
}

func (c *VllmConnection) HasAPIKey() bool {
	return c.apiKey != ""
}

func (c *VllmConnection) UsesTLS() bool {
	return strings.HasPrefix(c.baseURL, "https://")
}

func (c *VllmConnection) Request(ctx context.Context, method string, path string, authenticated bool, body string) (*http.Response, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.urlForPath(path), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated && c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return c.client.Do(req)
}

func (c *VllmConnection) Reachable(ctx context.Context) bool {
	resp, err := c.Request(ctx, http.MethodGet, "/health", false, "")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	discardProbeBody(resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func (c *VllmConnection) Version(ctx context.Context) (string, error) {
	c.versionOnce.Do(func() {
		resp, err := c.Request(ctx, http.MethodGet, "/version", false, "")
		if err != nil {
			c.versionErr = err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			c.versionErr = fmt.Errorf("vllm: /version returned HTTP %d", resp.StatusCode)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxVersionBody))
		if err != nil {
			c.versionErr = err
			return
		}
		var parsed struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Version != "" {
			c.version = parsed.Version
			return
		}
	})
	return c.version, c.versionErr
}

type ModelCard struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Root        string `json:"root"`
	Parent      string `json:"parent"`
	MaxModelLen int64  `json:"max_model_len"`
}

func (c *VllmConnection) Models(ctx context.Context) ([]ModelCard, error) {
	resp, err := c.Request(ctx, http.MethodGet, "/v1/models", true, "")
	if err != nil {
		return nil, fmt.Errorf("vllm: failed to list models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vllm: /v1/models returned HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []ModelCard `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("vllm: failed to parse /v1/models response: %w", err)
	}
	return parsed.Data, nil
}

func (c *VllmConnection) EndpointObservations(ctx context.Context) ([]EndpointObservation, error) {
	c.endpointsOnce.Do(func() {
		specs := DefaultEndpointSpecs()
		c.endpoints = make([]EndpointObservation, len(specs))
		workers := endpointProbeWorkers
		if len(specs) < workers {
			workers = len(specs)
		}
		jobs := make(chan int)
		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for idx := range jobs {
					c.endpoints[idx] = c.ProbeEndpoint(ctx, specs[idx])
				}
			}()
		}
		for i := range specs {
			select {
			case <-ctx.Done():
				c.endpointsErr = ctx.Err()
				close(jobs)
				wg.Wait()
				return
			case jobs <- i:
			}
		}
		close(jobs)
		wg.Wait()
	})
	return c.endpoints, c.endpointsErr
}

func (c *VllmConnection) ProbeEndpoint(ctx context.Context, spec EndpointSpec) EndpointObservation {
	spec.Method = strings.ToUpper(spec.Method)
	obs := EndpointObservation{Spec: spec}

	status, errText := c.probe(ctx, spec, false)
	obs.AnonymousStatusCode = status
	obs.AnonymousError = errText

	if c.apiKey != "" {
		status, errText = c.probe(ctx, spec, true)
		obs.AuthenticatedStatusCode = status
		obs.AuthenticatedError = errText
	}

	return obs
}

func (c *VllmConnection) CORS(ctx context.Context) (CORSObservation, error) {
	c.corsOnce.Do(func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodOptions, c.urlForPath("/"), nil)
		if err != nil {
			c.corsErr = err
			return
		}
		req.Header.Set("Origin", corsProbeOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		req.Header.Set("Access-Control-Request-Headers", corsProbeHeaders)

		resp, err := c.client.Do(req)
		if err != nil {
			c.corsErr = err
			return
		}
		defer resp.Body.Close()
		discardProbeBody(resp.Body)

		status := resp.StatusCode
		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		configured := allowOrigin != "" || resp.Header.Get("Access-Control-Allow-Methods") != "" || resp.Header.Get("Access-Control-Allow-Headers") != ""
		allowsAny := allowOrigin == "*"

		c.cors.StatusCode = &status
		c.cors.AllowOrigin = allowOrigin
		c.cors.Configured = &configured
		c.cors.AllowsAnyOrigin = &allowsAny
	})
	return c.cors, c.corsErr
}

func (c *VllmConnection) probe(ctx context.Context, spec EndpointSpec, authenticated bool) (*int, string) {
	body := spec.Body
	if body == "" && strings.EqualFold(spec.Method, http.MethodPost) {
		body = "{}"
	}
	resp, err := c.Request(ctx, spec.Method, spec.Path, authenticated, body)
	if err != nil {
		return nil, err.Error()
	}
	defer resp.Body.Close()
	discardProbeBody(resp.Body)

	status := resp.StatusCode
	return &status, ""
}

func discardProbeBody(body io.Reader) {
	_, _ = io.CopyN(io.Discard, body, maxProbeBody)
}

func (c *VllmConnection) urlForPath(path string) string {
	if path == "" || path == "/" {
		return c.baseURL + "/"
	}
	return c.baseURL + "/" + strings.TrimPrefix(path, "/")
}

func baseURLFromConfig(conf *inventory.Config) (string, error) {
	if conf.Options != nil {
		if raw := strings.TrimSpace(conf.Options[OptionBaseURL]); raw != "" {
			return normalizeBaseURL(raw)
		}
	}

	scheme := conf.Runtime
	if scheme == "" {
		scheme = "http"
	}
	host := conf.Host
	if host == "" {
		return "", fmt.Errorf("vllm: endpoint URL is required")
	}
	if conf.Port > 0 {
		host = net.JoinHostPort(host, strconv.Itoa(int(conf.Port)))
	}
	return normalizeBaseURL(scheme + "://" + host + conf.Path)
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("vllm: endpoint must use http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("vllm: endpoint URL must include a host")
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func apiKeyFromConfig(conf *inventory.Config) string {
	apiKey := os.Getenv("VLLM_API_KEY")
	if conf.Options != nil && conf.Options[OptionAPIKey] != "" {
		apiKey = conf.Options[OptionAPIKey]
	}
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			apiKey = string(cred.Secret)
		}
	}
	return strings.TrimSpace(apiKey)
}

func NewPostBody() string {
	return "{}"
}

func DefaultEndpointSpecs() []EndpointSpec {
	return []EndpointSpec{
		{Method: http.MethodGet, Path: "/docs", Category: "documentation"},
		{Method: http.MethodGet, Path: "/openapi.json", Category: "documentation"},
		{Method: http.MethodGet, Path: "/version", Category: "metadata"},
		{Method: http.MethodGet, Path: "/health", Category: "utility"},
		{Method: http.MethodGet, Path: "/ping", Category: "utility"},
		{Method: http.MethodGet, Path: "/load", Category: "utility"},
		{Method: http.MethodGet, Path: "/metrics", Category: "metrics"},
		{Method: http.MethodGet, Path: "/tokenizer_info", Category: "utility"},
		{Method: http.MethodPost, Path: "/tokenize", Category: "utility", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/detokenize", Category: "utility", Body: NewPostBody()},
		{Method: http.MethodGet, Path: "/v1/models", Category: "openai"},
		{Method: http.MethodPost, Path: "/v1/chat/completions", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/completions", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/embeddings", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/audio/transcriptions", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/audio/translations", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/messages", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/responses", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/score", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/v1/rerank", Category: "openai", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/invocations", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/inference/v1/generate", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/pooling", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/classify", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/score", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/rerank", Category: "custom-inference", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/pause", Category: "operational-control", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/resume", Category: "operational-control", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/scale_elastic_ep", Category: "operational-control", Body: NewPostBody()},
		{Method: http.MethodGet, Path: "/server_info", Category: "development"},
		{Method: http.MethodPost, Path: "/reset_prefix_cache", Category: "development", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/reset_mm_cache", Category: "development", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/reset_encoder_cache", Category: "development", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/sleep", Category: "development", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/wake_up", Category: "development", Body: NewPostBody()},
		{Method: http.MethodGet, Path: "/is_sleeping", Category: "development"},
		{Method: http.MethodPost, Path: "/collective_rpc", Category: "development", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/start_profile", Category: "profiler", Body: NewPostBody()},
		{Method: http.MethodPost, Path: "/stop_profile", Category: "profiler", Body: NewPostBody()},
	}
}

func ObservationPresent(obs EndpointObservation) bool {
	code := bestStatus(obs)
	return code != nil && *code != http.StatusNotFound && *code != http.StatusNotImplemented
}

func ObservationAnonymousAccessible(obs EndpointObservation) (bool, bool) {
	if obs.AnonymousStatusCode == nil {
		return false, false
	}
	switch *obs.AnonymousStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, true
	case http.StatusNotFound, http.StatusNotImplemented:
		return false, true
	default:
		if *obs.AnonymousStatusCode >= 500 {
			return false, false
		}
		return true, true
	}
}

// CategoryAnonymousAccessible aggregates the anonymous-access verdict across all
// observations in a functional category. It returns (true, true) as soon as any
// endpoint in the category is anonymously accessible; otherwise it returns
// (false, true) when at least one endpoint had a known verdict, and
// (false, false) when every endpoint in the category was unknown (so the caller
// can report null rather than a misleading "not exposed").
func CategoryAnonymousAccessible(observations []EndpointObservation, category string) (bool, bool) {
	known := false
	for _, obs := range observations {
		if obs.Spec.Category != category {
			continue
		}
		accessible, ok := ObservationAnonymousAccessible(obs)
		if !ok {
			continue
		}
		known = true
		if accessible {
			return true, true
		}
	}
	return false, known
}

func ObservationRequiresAuth(obs EndpointObservation) (bool, bool) {
	if obs.AnonymousStatusCode == nil || *obs.AnonymousStatusCode == http.StatusNotFound || *obs.AnonymousStatusCode >= 500 {
		return false, false
	}
	switch *obs.AnonymousStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return true, true
	default:
		return false, true
	}
}

func ObservationNotes(obs EndpointObservation) []string {
	notes := []string{}
	if obs.AnonymousError != "" {
		notes = append(notes, "anonymous probe error: "+obs.AnonymousError)
	}
	if obs.AuthenticatedError != "" {
		notes = append(notes, "authenticated probe error: "+obs.AuthenticatedError)
	}
	if obs.AnonymousStatusCode != nil {
		switch *obs.AnonymousStatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			notes = append(notes, "anonymous request was rejected by an authentication-like response")
		case http.StatusNotFound:
			notes = append(notes, "route was not observed")
		case http.StatusNotImplemented:
			notes = append(notes, "route is not implemented by the server")
		case http.StatusMethodNotAllowed:
			notes = append(notes, "route exists but rejected the probed HTTP method")
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			notes = append(notes, "anonymous request reached route validation")
		default:
			if *obs.AnonymousStatusCode < 400 {
				notes = append(notes, "anonymous request reached route successfully")
			}
		}
	}
	return notes
}

func bestStatus(obs EndpointObservation) *int {
	if obs.AuthenticatedStatusCode != nil && *obs.AuthenticatedStatusCode != http.StatusUnauthorized && *obs.AuthenticatedStatusCode != http.StatusForbidden {
		return obs.AuthenticatedStatusCode
	}
	return obs.AnonymousStatusCode
}
