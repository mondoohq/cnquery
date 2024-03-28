// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package transport

import (
	"net/http"

	"golang.org/x/sync/semaphore"
)

// MaxParallelConnHTTPTransport restricts the parallel connections that the client is doing upstream.
// This has many advantages:
// - we do not run into max ulimit issues because of parallel execution
// - we do not ddos our server in case something is wrong upstream
// - implementing this as http.RoundTripper has the advantage that the http timeout still applies and calls are canceled properly on the client-side
type MaxParallelConnHTTPTransport struct {
	transport     http.RoundTripper
	parallelConns *semaphore.Weighted
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *MaxParallelConnHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	err := t.parallelConns.Acquire(r.Context(), 1)
	if err != nil {
		return nil, err
	}
	defer t.parallelConns.Release(1)
	return t.transport.RoundTrip(r)
}

// NewMaxParallelConnTransport creates a transport with parallel HTTP connections
func NewMaxParallelConnTransport(transport http.RoundTripper, parallel int64) *MaxParallelConnHTTPTransport {
	return &MaxParallelConnHTTPTransport{
		transport:     transport,
		parallelConns: semaphore.NewWeighted(parallel),
	}
}
