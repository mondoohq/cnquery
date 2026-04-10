// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/providers/helm/provider"
)

var Config = plugin.Provider{
	Name:            "helm",
	ID:              "go.mondoo.com/mql/v13/providers/helm",
	Version:         "13.0.0",
	Maturity:        resources.MaturityExperimental,
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "helm",
			Use:   "helm PATH",
			Short: "a Helm chart directory or .tgz archive",
			Long: `Use the helm provider to query Helm charts, including chart metadata, templates, values, and rendered Kubernetes resources.

Examples:
  cnspec run helm ./my-chart -c "helm.charts { name version }"
  cnspec shell helm ./my-chart
`,
			MinArgs:   1,
			MaxArgs:   1,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
	AssetUrlTrees: []*inventory.AssetUrlBranch{
		{
			PathSegments: []string{"technology=iac", "category=helm"},
			Key:          "kind",
			Title:        "Kind",
			Values: map[string]*inventory.AssetUrlBranch{
				"chart": nil,
			},
		},
	},
}
