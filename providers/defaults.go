// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
//
// This file is auto-generated by 'make providers/defaults'

package providers

import "go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"

// DefaultProviders are useful when working in air-gapped environments
// to tell users what providers are used for common connections, when there
// is no other way to find out.
var DefaultProviders Providers = map[string]*Provider{
	"ansible": {
		Provider: &plugin.Provider{
			Name:            "ansible",
			ID:              "go.mondoo.com/cnquery/v9/providers/ansible",
			ConnectionTypes: []string{"ansible"},
			Connectors: []plugin.Connector{
				{
					Name:  "ansible",
					Use:   "ansible PATH",
					Short: "an Ansible playbook",
				},
			},
		},
	},

	"arista": {
		Provider: &plugin.Provider{
			Name:            "arista",
			ID:              "go.mondoo.com/cnquery/v9/providers/arista",
			ConnectionTypes: []string{"arista"},
			Connectors: []plugin.Connector{
				{
					Name:  "arista",
					Use:   "arista user@host",
					Short: "an Arista EOS device",
				},
			},
		},
	},

	"atlassian": {
		Provider: &plugin.Provider{
			Name:            "atlassian",
			ID:              "go.mondoo.com/cnquery/v9/providers/atlassian",
			ConnectionTypes: []string{"atlassian", "jira", "admin", "confluence", "scim"},
			Connectors: []plugin.Connector{
				{
					Name:  "atlassian",
					Use:   "atlassian",
					Short: "an Atlassian Cloud Jira, Confluence or Bitbucket instance",
				},
			},
		},
	},

	"aws": {
		Provider: &plugin.Provider{
			Name:            "aws",
			ID:              "go.mondoo.com/cnquery/v9/providers/aws",
			ConnectionTypes: []string{"aws", "ebs"},
			Connectors: []plugin.Connector{
				{
					Name:  "aws",
					Use:   "aws",
					Short: "an AWS account",
				},
			},
		},
	},

	"azure": {
		Provider: &plugin.Provider{
			Name:            "azure",
			ID:              "go.mondoo.com/cnquery/v9/providers/azure",
			ConnectionTypes: []string{"azure"},
			Connectors: []plugin.Connector{
				{
					Name:  "azure",
					Use:   "azure",
					Short: "an Azure subscription",
				},
			},
		},
	},

	"cloudformation": {
		Provider: &plugin.Provider{
			Name:            "cloudformation",
			ID:              "go.mondoo.com/cnquery/v9/providers/cloudformation",
			ConnectionTypes: []string{"cloudformation"},
			Connectors: []plugin.Connector{
				{
					Name:  "cloudformation",
					Use:   "cloudformation PATH",
					Short: "an AWS CloudFormation template or AWS SAM template",
				},
			},
		},
	},

	"cloudflare": {
		Provider: &plugin.Provider{
			Name:            "cloudflare",
			ID:              "go.mondoo.com/cnquery/v11/providers/cloudflare",
			ConnectionTypes: []string{"cloudflare"},
			Connectors: []plugin.Connector{
				{
					Name:  "cloudflare",
					Use:   "cloudflare",
					Short: "Cloudflare provider",
				},
			},
		},
	},

	"core": {
		Provider: &plugin.Provider{
			Name:            "core",
			ID:              "go.mondoo.com/cnquery/v9/providers/core",
			ConnectionTypes: []string(nil),
			Connectors:      []plugin.Connector{},
		},
	},

	"equinix": {
		Provider: &plugin.Provider{
			Name:            "equinix",
			ID:              "go.mondoo.com/cnquery/v9/providers/equinix",
			ConnectionTypes: []string{"equinix"},
			Connectors: []plugin.Connector{
				{
					Name:  "equinix",
					Use:   "equinix [org <org id>] [project <project-id>] [--token <token>]",
					Short: "an Equinix Metal organization",
				},
			},
		},
	},

	"gcp": {
		Provider: &plugin.Provider{
			Name:            "gcp",
			ID:              "go.mondoo.com/cnquery/v9/providers/gcp",
			ConnectionTypes: []string{"gcp", "gcp-snapshot"},
			Connectors: []plugin.Connector{
				{
					Name:  "gcp",
					Use:   "gcp",
					Short: "a Google Cloud project or folder",
				},
			},
		},
	},

	"github": {
		Provider: &plugin.Provider{
			Name:            "github",
			ID:              "go.mondoo.com/cnquery/v9/providers/github",
			ConnectionTypes: []string{"github"},
			Connectors: []plugin.Connector{
				{
					Name:  "github",
					Use:   "github",
					Short: "a GitHub organization or repository",
				},
			},
		},
	},

	"gitlab": {
		Provider: &plugin.Provider{
			Name:            "gitlab",
			ID:              "go.mondoo.com/cnquery/v9/providers/gitlab",
			ConnectionTypes: []string{"gitlab", "gitlab-group", "gitlab-project"},
			Connectors: []plugin.Connector{
				{
					Name:  "gitlab",
					Use:   "gitlab",
					Short: "a GitLab group or project",
				},
			},
		},
	},

	"google-workspace": {
		Provider: &plugin.Provider{
			Name:            "google-workspace",
			ID:              "go.mondoo.com/cnquery/v9/providers/google-workspace",
			ConnectionTypes: []string{"google-workspace"},
			Connectors: []plugin.Connector{
				{
					Name:  "google-workspace",
					Use:   "google-workspace [--credentials-path <credentials-path>] [--customer-id <customer-id>] [--impersonated-user-email <impersonated-user-email>]",
					Short: "a Google Workspace account",
				},
			},
		},
	},

	"ipmi": {
		Provider: &plugin.Provider{
			Name:            "ipmi",
			ID:              "go.mondoo.com/cnquery/v9/providers/ipmi",
			ConnectionTypes: []string{"ipmi"},
			Connectors: []plugin.Connector{
				{
					Name:  "ipmi",
					Use:   "ipmi user@host",
					Short: "an IPMI interface",
				},
			},
		},
	},

	"k8s": {
		Provider: &plugin.Provider{
			Name:            "k8s",
			ID:              "go.mondoo.com/cnquery/v9/providers/k8s",
			ConnectionTypes: []string{"k8s"},
			Connectors: []plugin.Connector{
				{
					Name:  "k8s",
					Use:   "k8s (optional MANIFEST path)",
					Short: "a Kubernetes cluster or local manifest file(s)",
				},
			},
		},
	},

	"ms365": {
		Provider: &plugin.Provider{
			Name:            "ms365",
			ID:              "go.mondoo.com/cnquery/v9/providers/ms365",
			ConnectionTypes: []string{"ms365"},
			Connectors: []plugin.Connector{
				{
					Name:  "ms365",
					Use:   "ms365",
					Short: "a Microsoft 365 account",
				},
			},
		},
	},

	"network": {
		Provider: &plugin.Provider{
			Name:            "network",
			ID:              "go.mondoo.com/cnquery/v9/providers/network",
			ConnectionTypes: []string{"host"},
			Connectors: []plugin.Connector{
				{
					Name:  "host",
					Use:   "host HOST",
					Short: "a remote host",
				},
			},
		},
	},

	"nmap": {
		Provider: &plugin.Provider{
			Name:            "nmap",
			ID:              "go.mondoo.com/cnquery/v9/providers/nmap",
			ConnectionTypes: []string{"nmap"},
			Connectors: []plugin.Connector{
				{
					Name:  "nmap",
					Use:   "nmap",
					Short: "a Nmap network scanner",
				},
			},
		},
	},

	"oci": {
		Provider: &plugin.Provider{
			Name:            "oci",
			ID:              "go.mondoo.com/cnquery/v9/providers/oci",
			ConnectionTypes: []string{"oci"},
			Connectors: []plugin.Connector{
				{
					Name:  "oci",
					Use:   "oci",
					Short: "an Oracle Cloud Infrastructure tenancy",
				},
			},
		},
	},

	"okta": {
		Provider: &plugin.Provider{
			Name:            "okta",
			ID:              "go.mondoo.com/cnquery/v9/providers/okta",
			ConnectionTypes: []string{"okta"},
			Connectors: []plugin.Connector{
				{
					Name:  "okta",
					Use:   "okta",
					Short: "an Okta organization",
				},
			},
		},
	},

	"opcua": {
		Provider: &plugin.Provider{
			Name:            "opcua",
			ID:              "go.mondoo.com/cnquery/v9/providers/opcua",
			ConnectionTypes: []string{"opcua"},
			Connectors: []plugin.Connector{
				{
					Name:  "opcua",
					Use:   "opcua [--endpoint <endpoint>]",
					Short: "an OPC UA device",
				},
			},
		},
	},

	"os": {
		Provider: &plugin.Provider{
			Name:            "os",
			ID:              "go.mondoo.com/cnquery/v9/providers/os",
			ConnectionTypes: []string{"local", "ssh", "tar", "docker-snapshot", "vagrant", "docker-image", "docker-container", "docker-file", "docker-registry", "container-registry", "registry-image", "filesystem"},
			Connectors: []plugin.Connector{
				{
					Name:  "local",
					Use:   "local",
					Short: "your local system",
				},

				{
					Name:  "ssh",
					Use:   "ssh user@host",
					Short: "a remote system via SSH",
				},

				{
					Name:  "winrm",
					Use:   "winrm user@host",
					Short: "a remote system via WinRM",
				},

				{
					Name:  "vagrant",
					Use:   "vagrant host",
					Short: "a Vagrant host",
				},

				{
					Name:  "container",
					Use:   "container",
					Short: "a running container or container image",
				},

				{
					Name:  "docker",
					Use:   "docker",
					Short: "a running Docker container or Docker image",
				},

				{
					Name:  "filesystem",
					Use:   "filesystem [flags]",
					Short: "a mounted file system target",
				},
			},
		},
	},

	"shodan": {
		Provider: &plugin.Provider{
			Name:            "shodan",
			ID:              "go.mondoo.com/cnquery/v9/providers/shodan",
			ConnectionTypes: []string{"shodan"},
			Connectors: []plugin.Connector{
				{
					Name:  "shodan",
					Use:   "shodan",
					Short: "a Shodan account",
				},
			},
		},
	},

	"slack": {
		Provider: &plugin.Provider{
			Name:            "slack",
			ID:              "go.mondoo.com/cnquery/v9/providers/slack",
			ConnectionTypes: []string{"slack"},
			Connectors: []plugin.Connector{
				{
					Name:  "slack",
					Use:   "slack",
					Short: "a Slack team",
				},
			},
		},
	},

	"snowflake": {
		Provider: &plugin.Provider{
			Name:            "snowflake",
			ID:              "go.mondoo.com/cnquery/v9/providers/snowflake",
			ConnectionTypes: []string{"snowflake"},
			Connectors: []plugin.Connector{
				{
					Name:  "snowflake",
					Use:   "snowflake",
					Short: "a Snowflake account",
				},
			},
		},
	},

	"terraform": {
		Provider: &plugin.Provider{
			Name:            "terraform",
			ID:              "go.mondoo.com/cnquery/v9/providers/terraform",
			ConnectionTypes: []string{"terraform-state", "terraform-plan", "terraform-hcl", "terraform-hcl-git"},
			Connectors: []plugin.Connector{
				{
					Name:  "terraform",
					Use:   "terraform PATH",
					Short: "a Terraform HCL file or directory",
				},
			},
		},
	},

	"vcd": {
		Provider: &plugin.Provider{
			Name:            "vcd",
			ID:              "go.mondoo.com/cnquery/v9/providers/vcd",
			ConnectionTypes: []string{"vcd"},
			Connectors: []plugin.Connector{
				{
					Name:  "vcd",
					Use:   "vcd [--user <user>] [--host <host>] [--organization <organization>] [--ask-pass] [--password <password>]",
					Short: "a VMware Cloud Director installation",
				},
			},
		},
	},

	"vsphere": {
		Provider: &plugin.Provider{
			Name:            "vsphere",
			ID:              "go.mondoo.com/cnquery/v9/providers/vsphere",
			ConnectionTypes: []string{"vsphere"},
			Connectors: []plugin.Connector{
				{
					Name:  "vsphere",
					Use:   "vsphere user@host",
					Short: "a VMware vSphere installation",
				},
			},
		},
	},
}
