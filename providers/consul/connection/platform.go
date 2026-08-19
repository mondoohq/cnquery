// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/url"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// PlatformIdConsulAgent prefixes the identifier of a Consul agent asset. The
// host is part of the identifier because each agent carries its own TLS and
// gossip configuration, so two agents in one datacenter are two assets.
const PlatformIdConsulAgent = "//platformid.api.mondoo.app/runtime/consul/host/"

// UnknownDatacenterLabel stands in for a datacenter that could not be read.
// The asset URL tree has a fixed depth, so the segment is always present and an
// unreadable datacenter needs a label rather than an empty string.
const UnknownDatacenterLabel = "unknown"

// NewConsulAgentPlatform describes a Consul agent asset. The datacenter is part
// of the technology path so agents group under the datacenter they serve.
func NewConsulAgentPlatform(host, datacenter string) *inventory.Platform {
	label := datacenter
	if label == "" {
		label = UnknownDatacenterLabel
	}
	segments := []string{"saas", "consul", "host", host, "datacenter", label}

	pf := &inventory.Platform{TechnologyUrlSegments: segments}
	PlatformByName("consul").Apply(pf)
	return pf
}

// NewConsulAgentIdentifier builds the platform ID of a Consul agent asset. The
// host is escaped, because it may carry characters that would otherwise read as
// a deeper path.
func NewConsulAgentIdentifier(host string) string {
	return PlatformIdConsulAgent + url.PathEscape(host)
}
