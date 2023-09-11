// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import "go.mondoo.com/cnquery/providers-sdk/v1/plugin"

const DefaultOsID = "go.mondoo.com/cnquery/providers/os"

var defaultRuntime *Runtime

func DefaultRuntime() *Runtime {
	if defaultRuntime == nil {
		defaultRuntime = Coordinator.NewRuntime()
	}
	return defaultRuntime
}

// DefaultProviders are useful when working in air-gapped environments
// to tell users what providers are used for common connections, when there
// is no other way to find out.
var DefaultProviders Providers = map[string]*Provider{
	"gcp": {
		Provider: &plugin.Provider{
			Name: "gcp",
			Connectors: []plugin.Connector{
				{
					Name:  "gcp",
					Short: "GCP Cloud",
				},
			},
		},
	},
	"ipmi": {
		Provider: &plugin.Provider{
			Name: "ipmi",
			Connectors: []plugin.Connector{
				{
					Name:  "ipmi",
					Short: "Ipmi",
				},
			},
		},
	},
	"arista": {
		Provider: &plugin.Provider{
			Name: "arista",
			Connectors: []plugin.Connector{
				{
					Name:  "arista",
					Short: "Arista EOS",
				},
			},
		},
	},
	"terraform": {
		Provider: &plugin.Provider{
			Name: "terraform",
			Connectors: []plugin.Connector{
				{
					Name:  "terraform",
					Short: "a terraform hcl file or directory.",
				},
			},
		},
	},
	"os": {
		Provider: &plugin.Provider{
			Name: "os",
			Connectors: []plugin.Connector{
				{
					Name:  "local",
					Short: "your local system",
				},
				{
					Name:  "ssh",
					Short: "a remote system via SSH",
				},
				{
					Name:  "winrm",
					Short: "a remote system via WinRM",
				},
				{
					Name:  "vagrant",
					Short: "a Vagrant host",
				},
				{
					Name:  "container",
					Short: "a running container or container image",
				},
				{
					Name:  "filesystem",
					Short: "a mounted file system target.",
				},
			},
		},
	},
	"vsphere": {
		Provider: &plugin.Provider{
			Name: "vsphere",
			Connectors: []plugin.Connector{
				{
					Name:  "vsphere",
					Short: "VMware vSphere",
				},
			},
		},
	},
	"google-workspace": {
		Provider: &plugin.Provider{
			Name: "google-workspace",
			Connectors: []plugin.Connector{
				{
					Name:  "google-workspace",
					Short: "Google Workspace",
				},
			},
		},
	},
	"opcua": {
		Provider: &plugin.Provider{
			Name: "opcua",
			Connectors: []plugin.Connector{
				{
					Name:  "opcua",
					Short: "OPC UA",
				},
			},
		},
	},
	"okta": {
		Provider: &plugin.Provider{
			Name: "okta",
			Connectors: []plugin.Connector{
				{
					Name:  "okta",
					Short: "Okta",
				},
			},
		},
	},
	"slack": {
		Provider: &plugin.Provider{
			Name: "slack",
			Connectors: []plugin.Connector{
				{
					Name:  "slack",
					Short: "slack team",
				},
			},
		},
	},
	"github": {
		Provider: &plugin.Provider{
			Name: "github",
			Connectors: []plugin.Connector{
				{
					Name:  "github",
					Short: "GitHub",
				},
			},
		},
	},
	"equinix": {
		Provider: &plugin.Provider{
			Name: "equinix",
			Connectors: []plugin.Connector{
				{
					Name:  "equinix",
					Short: "Equinix Metal",
				},
			},
		},
	},
	"k8s": {
		Provider: &plugin.Provider{
			Name: "k8s",
			Connectors: []plugin.Connector{
				{
					Name:  "k8s",
					Short: "a Kubernetes cluster or local manifest file(s).",
				},
			},
		},
	},
	"vcd": {
		Provider: &plugin.Provider{
			Name: "vcd",
			Connectors: []plugin.Connector{
				{
					Name:  "vcd",
					Short: "VMware Cloud Director",
				},
			},
		},
	},
	"aws": {
		Provider: &plugin.Provider{
			Name: "aws",
			Connectors: []plugin.Connector{
				{
					Name:  "aws",
					Short: "aws account",
				},
			},
		},
	},
	"gitlab": {
		Provider: &plugin.Provider{
			Name: "gitlab",
			Connectors: []plugin.Connector{
				{
					Name:  "gitlab",
					Short: "GitLab",
				},
			},
		},
	},
	"oci": {
		Provider: &plugin.Provider{
			Name: "oci",
			Connectors: []plugin.Connector{
				{
					Name:  "oci",
					Short: "Oracle Cloud Infrastructure",
				},
			},
		},
	},
	"network": {
		Provider: &plugin.Provider{
			Name: "network",
			Connectors: []plugin.Connector{
				{
					Name:  "host",
					Short: "a remote host",
				},
			},
		},
	},
	"ms365": {
		Provider: &plugin.Provider{
			Name: "ms365",
			Connectors: []plugin.Connector{
				{
					Name:  "ms365",
					Short: "ms365",
				},
			},
		},
	},
	"azure": {
		Provider: &plugin.Provider{
			Name: "azure",
			Connectors: []plugin.Connector{
				{
					Name:  "azure",
					Short: "azure",
				},
			},
		},
	},
}
