// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

const (
	Family = "nutanix"

	DiscoveryAll      = "all"
	DiscoveryAuto     = "auto"
	DiscoveryClusters = "clusters"
	DiscoveryNodes    = "nodes"
)

const (
	platformIdPrismCentral = "//platformid.api.mondoo.app/runtime/nutanix/endpoint/"
	platformIdCluster      = "//platformid.api.mondoo.app/runtime/nutanix/cluster/"
	platformIdNode         = "//platformid.api.mondoo.app/runtime/nutanix/node/"
)

// ClusterID returns the cluster UUID this connection is scoped to, or "" when
// the connection targets Prism Central as a whole.
func (c *NutanixConnection) ClusterID() string {
	return c.Conf.Options["cluster-id"]
}

// NodeID returns the host UUID this connection is scoped to, or "" when the
// connection is not scoped to a single node.
func (c *NutanixConnection) NodeID() string {
	return c.Conf.Options["node-id"]
}

// PlatformInfo returns the platform for the asset this connection represents,
// derived from the scope encoded in the connection options.
func (c *NutanixConnection) PlatformInfo() *inventory.Platform {
	if id := c.NodeID(); id != "" {
		return NewNodePlatform(id)
	}
	if id := c.ClusterID(); id != "" {
		return NewClusterPlatform(id)
	}
	return NewPrismCentralPlatform(c.endpoint)
}

// PlatformIDs returns the stable platform identifiers for the asset this
// connection represents.
func (c *NutanixConnection) PlatformIDs() []string {
	if id := c.NodeID(); id != "" {
		return []string{NewNodeIdentifier(id)}
	}
	if id := c.ClusterID(); id != "" {
		return []string{NewClusterIdentifier(id)}
	}
	return []string{NewPrismCentralIdentifier(c.endpoint)}
}

func NewPrismCentralPlatform(endpoint string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "nutanix-prism-central",
		Title:                 "Nutanix Prism Central",
		Family:                []string{Family},
		Kind:                  "api",
		Runtime:               "nutanix",
		TechnologyUrlSegments: []string{"virtualization", "nutanix", "prism-central", endpoint},
	}
}

func NewClusterPlatform(clusterId string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "nutanix-cluster",
		Title:                 "Nutanix Cluster",
		Family:                []string{Family},
		Kind:                  "api",
		Runtime:               "nutanix",
		TechnologyUrlSegments: []string{"virtualization", "nutanix", "cluster", clusterId},
	}
}

func NewNodePlatform(nodeId string) *inventory.Platform {
	return &inventory.Platform{
		Name:                  "nutanix-node",
		Title:                 "Nutanix Node",
		Family:                []string{Family},
		Kind:                  inventory.AssetKindBaremetal,
		Runtime:               "nutanix",
		TechnologyUrlSegments: []string{"virtualization", "nutanix", "node", nodeId},
	}
}

func NewPrismCentralIdentifier(endpoint string) string {
	return platformIdPrismCentral + endpoint
}

func NewClusterIdentifier(clusterId string) string {
	return platformIdCluster + clusterId
}

func NewNodeIdentifier(nodeId string) string {
	return platformIdNode + nodeId
}
