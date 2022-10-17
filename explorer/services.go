package explorer

import (
	"net/http"

	"go.mondoo.com/ranger-rpc"
	"golang.org/x/sync/semaphore"
)

type ResolvedVersion string

const (
	V2Code ResolvedVersion = "v2"
)

var globalEmpty = &Empty{}

type Services struct {
	QueryHub
	QueryConductor
}

// LocalServices is an implementation of the explorer for a local execution.
// It has an optional upstream-handler embedded. If a local service does not
// yield results for a request, and the upstream handler is defined, it will
// be used instead.
type LocalServices struct {
	DataLake  DataLake
	Upstream  *Services
	Incognito bool
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, uuid string) *LocalServices {
	return &LocalServices{
		DataLake:  datalake,
		Upstream:  nil,
		Incognito: false,
	}
}

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin) (*Services, error) {
	client := ranger.DefaultHttpClient()
	// restrict parallel upstream connections to two connections
	client.Transport = NewMaxParallelConnTransport(client.Transport, 2)

	queryHub, err := NewQueryHubClient(addr, client, auth...)
	if err != nil {
		return nil, err
	}

	queryConductor, err := NewQueryConductorClient(addr, client, auth...)
	if err != nil {
		return nil, err
	}

	return &Services{
		QueryHub:       queryHub,
		QueryConductor: queryConductor,
	}, nil
}

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
