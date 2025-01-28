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
	Version:         "11.1.47",
	ConnectionTypes: []string{provider.ConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:    "k8s",
			Aliases: []string{"kubernetes"},
			Use:     "k8s (optional MANIFEST path)",
			Short:   "a Kubernetes cluster or local manifest file(s)",
			Long: `Use the k8s provider to query Kubernetes resources, including clusters, pods, services, containers, manifests, and more.

Requirement:
  To query or scan a Kubernetes cluster, you must install kubectl on your workstation. To learn how, read https://kubernetes.io/docs/tasks/tools/. 

Examples:
  cnquery shell k8s
  cnspec scan k8s
  cnspec <MANIFEST-FILE>
`,
			MinArgs: 0,
			MaxArgs: 1,
			Discovery: []string{
				resources.DiscoveryAdmissionReviews,
				resources.DiscoveryClusters,
				resources.DiscoveryContainerImages,
				resources.DiscoveryCronJobs,
				resources.DiscoveryDaemonSets,
				resources.DiscoveryDeployments,
				resources.DiscoveryIngresses,
				resources.DiscoveryJobs,
				resources.DiscoveryNamespaces,
				resources.DiscoveryPods,
				resources.DiscoveryReplicaSets,
				resources.DiscoveryServices,
				resources.DiscoveryStatefulSets,
			},
			Flags: []plugin.Flag{
				{
					Long:    "context",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Target a Kubernetes context",
				},
				{
					Long:    "namespaces-exclude",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Filter out Kubernetes objects in the matching namespaces",
				},
				{
					Long:    "namespaces",
					Type:    plugin.FlagType_String,
					Default: "",
					Desc:    "Only include Kubernetes object in the matching namespaces",
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
		{
			PathSegments: []string{"technology=iac", "category=k8s-manifest"},
		},
	},
}
