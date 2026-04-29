// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/vllm/connection"
)

type mqlVllmEndpointInternal struct {
	once sync.Once
	obs  connection.EndpointObservation
}

func (r *mqlVllm) id() (string, error) {
	conn, err := vllmConnection(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return conn.BaseURL(), nil
}

func (r *mqlVllm) server() (*mqlVllmServer, error) {
	res, err := CreateResource(r.MqlRuntime, "vllm.server", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlVllmServer), nil
}

func (r *mqlVllm) endpoints() ([]any, error) {
	conn, err := vllmConnection(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	specs := connection.DefaultEndpointSpecs()
	res := make([]any, 0, len(specs))
	observations, err := conn.EndpointObservations(context.Background())
	if err != nil {
		return nil, err
	}
	for _, obs := range observations {
		endpoint, err := CreateResource(r.MqlRuntime, "vllm.endpoint", map[string]*llx.RawData{
			"path":   llx.StringData(obs.Spec.Path),
			"method": llx.StringData(obs.Spec.Method),
		})
		if err != nil {
			return nil, err
		}
		if e, ok := endpoint.(*mqlVllmEndpoint); ok {
			primed := obs
			e.once.Do(func() {
				e.obs = primed
			})
		}
		res = append(res, endpoint)
	}
	return res, nil
}

func (r *mqlVllm) metrics() (*mqlVllmMetrics, error) {
	res, err := CreateResource(r.MqlRuntime, "vllm.metrics", nil)
	if err != nil {
		return nil, err
	}
	return res.(*mqlVllmMetrics), nil
}

func (r *mqlVllm) version() (string, error) {
	conn, err := vllmConnection(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	version, err := conn.Version(context.Background())
	if err != nil {
		r.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	if version == "" {
		r.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return version, nil
}

func (s *mqlVllmServer) id() (string, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return "", err
	}
	return conn.BaseURL(), nil
}

func (s *mqlVllmServer) baseUrl() (string, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return "", err
	}
	return conn.BaseURL(), nil
}

func (s *mqlVllmServer) reachable() (bool, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return false, err
	}
	return conn.Reachable(context.Background()), nil
}

func (s *mqlVllmServer) version() (string, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return "", err
	}
	version, err := conn.Version(context.Background())
	if err != nil || version == "" {
		s.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return version, nil
}

func (s *mqlVllmServer) docsExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(s.MqlRuntime, http.MethodGet, "/docs")
	if err != nil {
		return false, err
	}
	if !known {
		s.DocsExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) openapiExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(s.MqlRuntime, http.MethodGet, "/openapi.json")
	if err != nil {
		return false, err
	}
	if !known {
		s.OpenapiExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) metricsExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(s.MqlRuntime, http.MethodGet, "/metrics")
	if err != nil {
		return false, err
	}
	if !known {
		s.MetricsExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) loadEndpointExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(s.MqlRuntime, http.MethodGet, "/load")
	if err != nil {
		return false, err
	}
	if !known {
		s.LoadEndpointExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) tokenizerInfoExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(s.MqlRuntime, http.MethodGet, "/tokenizer_info")
	if err != nil {
		return false, err
	}
	if !known {
		s.TokenizerInfoExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) devEndpointsExposed() (bool, error) {
	accessible, known, err := categoryAnonymousAccessibleKnown(s.MqlRuntime, "development")
	if err != nil {
		return false, err
	}
	if !known {
		s.DevEndpointsExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) profilerEndpointsExposed() (bool, error) {
	accessible, known, err := categoryAnonymousAccessibleKnown(s.MqlRuntime, "profiler")
	if err != nil {
		return false, err
	}
	if !known {
		s.ProfilerEndpointsExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (s *mqlVllmServer) corsConfigured() (bool, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return false, err
	}
	obs, err := conn.CORS(context.Background())
	if err != nil || obs.Configured == nil {
		s.CorsConfigured.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return *obs.Configured, nil
}

func (s *mqlVllmServer) corsAllowsAnyOrigin() (bool, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return false, err
	}
	obs, err := conn.CORS(context.Background())
	if err != nil || obs.AllowsAnyOrigin == nil {
		s.CorsAllowsAnyOrigin.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return *obs.AllowsAnyOrigin, nil
}

func (s *mqlVllmServer) usesTls() (bool, error) {
	conn, err := vllmConnection(s.MqlRuntime)
	if err != nil {
		return false, err
	}
	return conn.UsesTLS(), nil
}

func initVllmEndpoint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["path"]; !ok {
		return nil, nil, errors.New("vllm.endpoint requires path")
	}
	if _, ok := args["method"]; !ok {
		args["method"] = llx.StringData(http.MethodGet)
	}
	if method, ok := args["method"]; ok {
		args["method"] = llx.StringData(strings.ToUpper(method.Value.(string)))
	}
	return args, nil, nil
}

func (e *mqlVllmEndpoint) id() (string, error) {
	return e.Method.Data + " " + e.Path.Data, nil
}

func (e *mqlVllmEndpoint) category() (string, error) {
	obs, err := e.observation()
	if err != nil {
		return "", err
	}
	return obs.Spec.Category, nil
}

func (e *mqlVllmEndpoint) present() (bool, error) {
	obs, err := e.observation()
	if err != nil {
		return false, err
	}
	return connection.ObservationPresent(obs), nil
}

func (e *mqlVllmEndpoint) anonymousStatusCode() (int64, error) {
	obs, err := e.observation()
	if err != nil {
		return 0, err
	}
	if obs.AnonymousStatusCode == nil {
		e.AnonymousStatusCode.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return int64(*obs.AnonymousStatusCode), nil
}

func (e *mqlVllmEndpoint) authenticatedStatusCode() (int64, error) {
	obs, err := e.observation()
	if err != nil {
		return 0, err
	}
	if obs.AuthenticatedStatusCode == nil {
		e.AuthenticatedStatusCode.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return int64(*obs.AuthenticatedStatusCode), nil
}

func (e *mqlVllmEndpoint) anonymousAccessible() (bool, error) {
	obs, err := e.observation()
	if err != nil {
		return false, err
	}
	val, known := connection.ObservationAnonymousAccessible(obs)
	if !known {
		e.AnonymousAccessible.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return val, nil
}

func (e *mqlVllmEndpoint) requiresAuth() (bool, error) {
	obs, err := e.observation()
	if err != nil {
		return false, err
	}
	val, known := connection.ObservationRequiresAuth(obs)
	if !known {
		e.RequiresAuth.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return val, nil
}

func (e *mqlVllmEndpoint) notes() ([]any, error) {
	obs, err := e.observation()
	if err != nil {
		return nil, err
	}
	notes := connection.ObservationNotes(obs)
	res := make([]any, len(notes))
	for i := range notes {
		res[i] = notes[i]
	}
	return res, nil
}

func (e *mqlVllmEndpoint) observation() (connection.EndpointObservation, error) {
	var err error
	e.once.Do(func() {
		conn, connErr := vllmConnection(e.MqlRuntime)
		if connErr != nil {
			err = connErr
			return
		}
		spec := endpointSpecFor(e.Method.Data, e.Path.Data)
		e.obs = conn.ProbeEndpoint(context.Background(), spec)
	})
	return e.obs, err
}

func (m *mqlVllmMetrics) id() (string, error) {
	conn, err := vllmConnection(m.MqlRuntime)
	if err != nil {
		return "", err
	}
	return conn.BaseURL() + "/metrics", nil
}

func (m *mqlVllmMetrics) prometheusExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(m.MqlRuntime, http.MethodGet, "/metrics")
	if err != nil {
		return false, err
	}
	if !known {
		m.PrometheusExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (m *mqlVllmMetrics) loadEndpointExposed() (bool, error) {
	accessible, known, err := endpointAnonymousAccessibleKnown(m.MqlRuntime, http.MethodGet, "/load")
	if err != nil {
		return false, err
	}
	if !known {
		m.LoadEndpointExposed.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return accessible, nil
}

func (m *mqlVllmMetrics) loadTrackingVisible() (bool, error) {
	obs, err := endpointObservation(m.MqlRuntime, http.MethodGet, "/load")
	if err != nil {
		return false, err
	}
	accessible, known := connection.ObservationAnonymousAccessible(obs)
	if !known {
		m.LoadTrackingVisible.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return accessible, nil
}

func vllmConnection(runtime *plugin.Runtime) (*connection.VllmConnection, error) {
	conn, ok := runtime.Connection.(*connection.VllmConnection)
	if !ok {
		return nil, errors.New("invalid connection for vllm provider")
	}
	return conn, nil
}

func endpointAnonymousAccessibleKnown(runtime *plugin.Runtime, method string, path string) (bool, bool, error) {
	obs, err := endpointObservation(runtime, method, path)
	if err != nil {
		return false, false, err
	}
	val, known := connection.ObservationAnonymousAccessible(obs)
	return val, known, nil
}

func categoryAnonymousAccessibleKnown(runtime *plugin.Runtime, category string) (bool, bool, error) {
	conn, err := vllmConnection(runtime)
	if err != nil {
		return false, false, err
	}
	observations, err := conn.EndpointObservations(context.Background())
	if err != nil {
		return false, false, err
	}
	knownCount := 0
	for _, obs := range observations {
		if obs.Spec.Category != category {
			continue
		}
		accessible, known := connection.ObservationAnonymousAccessible(obs)
		if !known {
			continue
		}
		knownCount++
		if accessible {
			return true, true, nil
		}
	}
	return false, knownCount > 0, nil
}

func endpointObservation(runtime *plugin.Runtime, method string, path string) (connection.EndpointObservation, error) {
	conn, err := vllmConnection(runtime)
	if err != nil {
		return connection.EndpointObservation{}, err
	}
	observations, err := conn.EndpointObservations(context.Background())
	if err != nil {
		return connection.EndpointObservation{}, err
	}
	method = strings.ToUpper(method)
	for _, obs := range observations {
		if obs.Spec.Method == method && obs.Spec.Path == path {
			return obs, nil
		}
	}
	return conn.ProbeEndpoint(context.Background(), endpointSpecFor(method, path)), nil
}

func endpointSpecFor(method string, path string) connection.EndpointSpec {
	method = strings.ToUpper(method)
	for _, spec := range connection.DefaultEndpointSpecs() {
		if spec.Method == method && spec.Path == path {
			return spec
		}
	}
	body := ""
	if method == http.MethodPost {
		body = connection.NewPostBody()
	}
	return connection.EndpointSpec{
		Method:   method,
		Path:     path,
		Category: "custom",
		Body:     body,
	}
}
