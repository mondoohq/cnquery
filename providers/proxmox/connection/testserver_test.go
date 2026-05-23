// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakePVE spins up an httptest.Server that mimics the Proxmox API
// envelope (`{"data": ...}`) and lets tests script per-path responses.
// Use t.Cleanup to register Close, and call newConnection with the
// server URL.
type fakePVE struct {
	t        *testing.T
	server   *httptest.Server
	routes   map[string]fakeRoute
	requests []string // record of paths hit, for assertions
}

type fakeRoute struct {
	status int
	body   any    // marshaled into the `data` field of the envelope
	raw    string // when set, returned verbatim instead of using `body`
}

func newFakePVE(t *testing.T) *fakePVE {
	t.Helper()
	f := &fakePVE{t: t, routes: map[string]fakeRoute{}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Proxmox responses are always /api2/json/<path>; strip the prefix.
		path := r.URL.Path
		const prefix = "/api2/json"
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			path = path[len(prefix):]
		}
		// Include the query string so tests can register routes like
		// "/nodes/pve/disks/smart?disk=/dev/sda".
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		f.requests = append(f.requests, path)
		route, ok := f.routes[path]
		if !ok {
			http.Error(w, "no route", http.StatusNotFound)
			return
		}
		if route.status >= 400 && route.body == nil && route.raw == "" {
			http.Error(w, "forced error", route.status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if route.status != 0 {
			w.WriteHeader(route.status)
		}
		if route.raw != "" {
			_, _ = w.Write([]byte(route.raw))
			return
		}
		payload := map[string]any{"data": route.body}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(f.server.Close)
	return f
}

// route registers a successful response body for the given API path.
// The body is wrapped in the Proxmox `{"data": ...}` envelope.
func (f *fakePVE) route(path string, body any) {
	f.routes[path] = fakeRoute{body: body}
}

// errorRoute registers a path that returns an HTTP error status with an
// optional message in the envelope (mirroring how the real API reports
// permission/not-found errors).
func (f *fakePVE) errorRoute(path string, status int, message string) {
	if message == "" {
		f.routes[path] = fakeRoute{status: status}
		return
	}
	f.routes[path] = fakeRoute{
		status: status,
		raw:    `{"data":null,"message":` + jsonQuote(message) + `}`,
	}
}

// rawRoute lets a test bypass the envelope (useful for malformed JSON
// regression tests).
func (f *fakePVE) rawRoute(path string, status int, raw string) {
	f.routes[path] = fakeRoute{status: status, raw: raw}
}

func (f *fakePVE) conn() *PveConnection {
	// The token doesn't have to be valid; the fake server doesn't check.
	return NewConnection(0, f.server.URL, "PVEAPIToken=root@pam!test=fake", true)
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
