// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"net/http"

	"go.mondoo.com/cnquery/v10/explorer/resources"
	"go.mondoo.com/cnquery/v10/explorer/transport"
	llx "go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/ranger-rpc"
)

type ResolvedVersion string

const (
	V2Code ResolvedVersion = "v2"
)

var globalEmpty = &Empty{}

type Services struct {
	QueryHub
	QueryConductor
	resources.ResourcesExplorer
}

// LocalServices is an implementation of the explorer for a local execution.
// It has an optional upstream-handler embedded. If a local service does not
// yield results for a request, and the upstream handler is defined, it will
// be used instead.
type LocalServices struct {
	DataLake  DataLake
	Upstream  *Services
	Incognito bool
	runtime   llx.Runtime
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, uuid string, runtime llx.Runtime) *LocalServices {
	return &LocalServices{
		DataLake:  datalake,
		Upstream:  nil,
		Incognito: false,
		runtime:   runtime,
	}
}

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin, httpClient *http.Client) (*Services, error) {
	if httpClient == nil {
		httpClient = ranger.DefaultHttpClient()
	}
	// restrict parallel upstream connections to two connections
	httpClient.Transport = transport.NewMaxParallelConnTransport(httpClient.Transport, 2)

	queryHub, err := NewQueryHubClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	queryConductor, err := NewQueryConductorClient(addr, httpClient, auth...)
	if err != nil {
		return nil, err
	}

	return &Services{
		QueryHub:       queryHub,
		QueryConductor: queryConductor,
	}, nil
}
