// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"net/http"

	ranger "go.mondoo.com/ranger-rpc"
)

// NewRemoteServices creates a new ResourcesExplorer client for upstream services
func NewRemoteServices(addr string, plugins []ranger.ClientPlugin, httpClient *http.Client) (ResourcesExplorer, error) {
	return NewResourcesExplorerClient(addr, httpClient, plugins...)
}
