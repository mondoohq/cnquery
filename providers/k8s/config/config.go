// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/k8s/resources"
)

var Config = plugin.Provider{
	Name:    "k8s",
	ID:      "go.mondoo.com/cnquery/providers/k8s",
	Version: "9.0.0",
	Connectors: []plugin.Connector{
		{
			Name:    "k8s",
			Aliases: []string{"kubernetes"},
			Use:     "k8s (optional MANIFEST path)",
			Short:   "a Kubernetes cluster or local manifest file(s).",
			MinArgs: 0,
			MaxArgs: 1,
			Discovery: []string{
				resources.DiscoveryPods,
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
			},
		},
	},
}
