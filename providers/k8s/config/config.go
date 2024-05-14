// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/k8s/provider"
	"go.mondoo.com/cnquery/v11/providers/k8s/resources"
)

var Config = plugin.Provider{
	Name:            "k8s",
	ID:              "go.mondoo.com/cnquery/v9/providers/k8s",
	Version:         "11.0.6",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:    "k8s",
			Aliases: []string{"kubernetes"},
			Use:     "k8s (optional MANIFEST path)",
			Short:   "a Kubernetes cluster or local manifest file(s)",
			MinArgs: 0,
			MaxArgs: 1,
			Discovery: []string{
				resources.DiscoveryAll,
				resources.DiscoveryAuto,
				resources.DiscoveryClusters,
				resources.DiscoveryPods,
				resources.DiscoveryJobs,
				resources.DiscoveryCronJobs,
				resources.DiscoveryStatefulSets,
				resources.DiscoveryDeployments,
				resources.DiscoveryReplicaSets,
				resources.DiscoveryDaemonSets,
				resources.DiscoveryContainerImages,
				resources.DiscoveryAdmissionReviews,
				resources.DiscoveryIngresses,
				resources.DiscoveryNamespaces,
			},
			Flags: []plugin.Flag{
				{
					Long:    "context",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Target a Kubernetes context.",
				},
				{
					Long:    "namespaces-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Filter out Kubernetes objects in the matching namespaces.",
				},
				{
					Long:    "namespaces",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Only include Kubernetes object in the matching namespaces.",
				},
				{
					Long:    "container-proxy",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "HTTP proxy to use for container pulls",
				},
			},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=k8s"},
			Key:          "platform",
			Title:        "Platform",
			Values: map[string]*inventory.AssetUrlBranch{
				"*": nil,
			},
		},
	},
}
