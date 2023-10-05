// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
)

const DefaultOsID = "go.mondoo.com/cnquery/v9/providers/os"

var defaultRuntime *Runtime

func DefaultRuntime() *Runtime {
	if defaultRuntime == nil {
		defaultRuntime = Coordinator.NewRuntime()
	}
	return defaultRuntime
}

func SetDefaultRuntime(rt *Runtime) error {
	if rt == nil {
		return errors.New("attempted to set default runtime to null")
	}
	defaultRuntime = rt
	return nil
}

// DefaultProviders are useful when working in air-gapped environments
// to tell users what providers are used for common connections, when there
// is no other way to find out.
var DefaultProviders Providers = map[string]*Provider{
	"arista": {
		Provider: &plugin.Provider{
			Name:            "arista",
			ConnectionTypes: []string{"arista"},
			Connectors: []plugin.Connector{
				{
					Name:  "arista",
					Short: "Arista EOS",
				},
			},
		},
	},
	"aws": {
		Provider: &plugin.Provider{
			Name:            "aws",
			ConnectionTypes: []string{"aws", "ebs"},
			Connectors: []plugin.Connector{
				{
					Name:  "aws",
					Short: "aws account",
				},
			},
		},
	},
	"azure": {
		Provider: &plugin.Provider{
			Name:            "azure",
			ConnectionTypes: []string{"azure"},
			Connectors: []plugin.Connector{
				{
					Name:  "azure",
					Short: "azure",
				},
			},
		},
	},
	"core": {
		Provider: &plugin.Provider{
			Name:            "core",
			ConnectionTypes: []string(nil),
			Connectors:      []plugin.Connector{},
		},
	},
	"equinix": {
		Provider: &plugin.Provider{
			Name:            "equinix",
			ConnectionTypes: []string{"equinix"},
			Connectors: []plugin.Connector{
				{
					Name:  "equinix",
					Short: "Equinix Metal",
				},
			},
		},
	},
	"gcp": {
		Provider: &plugin.Provider{
			Name:            "gcp",
			ConnectionTypes: []string{"gcp", "gcp-snapshot"},
			Connectors: []plugin.Connector{
				{
					Name:  "gcp",
					Short: "GCP Cloud",
				},
			},
		},
	},
	"github": {
		Provider: &plugin.Provider{
			Name:            "github",
			ConnectionTypes: []string{"github"},
			Connectors: []plugin.Connector{
				{
					Name:  "github",
					Short: "GitHub",
				},
			},
		},
	},
	"gitlab": {
		Provider: &plugin.Provider{
			Name:            "gitlab",
			ConnectionTypes: []string{"gitlab", "gitlab-group", "gitlab-project"},
			Connectors: []plugin.Connector{
				{
					Name:  "gitlab",
					Short: "GitLab",
				},
			},
		},
	},
	"google-workspace": {
		Provider: &plugin.Provider{
			Name:            "google-workspace",
			ConnectionTypes: []string{"google-workspace"},
			Connectors: []plugin.Connector{
				{
					Name:  "google-workspace",
					Short: "Google Workspace",
				},
			},
		},
	},
	"ipmi": {
		Provider: &plugin.Provider{
			Name:            "ipmi",
			ConnectionTypes: []string{"ipmi"},
			Connectors: []plugin.Connector{
				{
					Name:  "ipmi",
					Short: "Ipmi",
				},
			},
		},
	},
	"k8s": {
		Provider: &plugin.Provider{
			Name:            "k8s",
			ConnectionTypes: []string{"k8s"},
			Connectors: []plugin.Connector{
				{
					Name:  "k8s",
					Short: "a Kubernetes cluster or local manifest file(s).",
				},
			},
		},
	},
	"ms365": {
		Provider: &plugin.Provider{
			Name:            "ms365",
			ConnectionTypes: []string{"ms365"},
			Connectors: []plugin.Connector{
				{
					Name:  "ms365",
					Short: "ms365",
				},
			},
		},
	},
	"network": {
		Provider: &plugin.Provider{
			Name:            "network",
			ConnectionTypes: []string{"host"},
			Connectors: []plugin.Connector{
				{
					Name:  "host",
					Short: "a remote host",
				},
			},
		},
	},
	"oci": {
		Provider: &plugin.Provider{
			Name:            "oci",
			ConnectionTypes: []string{"oci"},
			Connectors: []plugin.Connector{
				{
					Name:  "oci",
					Short: "Oracle Cloud Infrastructure",
				},
			},
		},
	},
	"okta": {
		Provider: &plugin.Provider{
			Name:            "okta",
			ConnectionTypes: []string{"okta"},
			Connectors: []plugin.Connector{
				{
					Name:  "okta",
					Short: "Okta",
				},
			},
		},
	},
	"opcua": {
		Provider: &plugin.Provider{
			Name:            "opcua",
			ConnectionTypes: []string{"opcua"},
			Connectors: []plugin.Connector{
				{
					Name:  "opcua",
					Short: "OPC UA",
				},
			},
		},
	},
	"os": {
		Provider: &plugin.Provider{
			Name:            "os",
			ConnectionTypes: []string{"local", "ssh", "tar", "docker-snapshot", "vagrant", "docker-image", "docker-container", "docker-registry", "container-registry", "registry-image", "filesystem"},
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
					Name:  "docker",
					Short: "a running docker or docker image",
				},
				{
					Name:  "filesystem",
					Short: "a mounted file system target.",
				},
			},
		},
	},
	"slack": {
		Provider: &plugin.Provider{
			Name:            "slack",
			ConnectionTypes: []string{"slack"},
			Connectors: []plugin.Connector{
				{
					Name:  "slack",
					Short: "slack team",
				},
			},
		},
	},
	"terraform": {
		Provider: &plugin.Provider{
			Name:            "terraform",
			ConnectionTypes: []string{"terraform-state", "terraform-plan", "terraform-hcl", "terraform-hcl-git"},
			Connectors: []plugin.Connector{
				{
					Name:  "terraform",
					Short: "a terraform hcl file or directory.",
				},
			},
		},
	},
	"vcd": {
		Provider: &plugin.Provider{
			Name:            "vcd",
			ConnectionTypes: []string{"vcd"},
			Connectors: []plugin.Connector{
				{
					Name:  "vcd",
					Short: "VMware Cloud Director",
				},
			},
		},
	},
	"vsphere": {
		Provider: &plugin.Provider{
			Name:            "vsphere",
			ConnectionTypes: []string{"vsphere"},
			Connectors: []plugin.Connector{
				{
					Name:  "vsphere",
					Short: "VMware vSphere",
				},
			},
		},
	},
}
