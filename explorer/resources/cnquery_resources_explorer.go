// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// The resources explorer allows us to quickly interact with an upstream
// resources store. It is kept separate from the explorer

package resources

import (
	"net/http"

	"go.mondoo.com/cnquery/v10/explorer/transport"
	ranger "go.mondoo.com/ranger-rpc"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_resources_explorer.proto

// NewRemoteServices initializes a services struct with a remote endpoint
func NewRemoteServices(addr string, auth []ranger.ClientPlugin, httpClient *http.Client) (ResourcesExplorer, error) {
	if httpClient == nil {
		httpClient = ranger.DefaultHttpClient()
	}
	// restrict parallel upstream connections to two connections
	httpClient.Transport = transport.NewMaxParallelConnTransport(httpClient.Transport, 2)

	return NewResourcesExplorerClient(addr, httpClient, auth...)
}
