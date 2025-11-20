// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ipinfo/provider"
)

var Config = plugin.Provider{
	Name:            "ipinfo",
	ID:              "go.mondoo.com/cnquery/v12/providers/ipinfo",
	Version:         "12.10.0",
	ConnectionTypes: []string{provider.DefaultConnectionType},
	Connectors: []plugin.Connector{
		{
			Name:  "ipinfo",
			Use:   "ipinfo",
			Short: "IP information from ipinfo.io",
			Long: `Use the ipinfo provider to query IP address information from ipinfo.io, including the IP address, hostname, and whether the IP address is a bogon.

Examples:
  cnquery shell ipinfo
  cnquery run ipinfo -c "ipinfo(ip('8.8.8.8')){*}"
  cnquery run ipinfo -c "ipinfo(){*}"  # Query your public IP"

Notes:
  - Pass an IP address to query information about that specific IP: ipinfo(ip("1.1.1.1"))
  - Pass no arguments (empty IP) to query your machine's public IP: ipinfo()
  - The bogon field indicates whether the returned IP is a private, link-local, or otherwise non-routable address. When bogon is true, the returned IP is the same as the requested IP.
  - Set IPINFO_TOKEN environment variable to use the authenticated ipinfo.io API (this is optional, free API is used by default).
`,
			Discovery: []string{},
			Flags:     []plugin.Flag{},
		},
	},
}
