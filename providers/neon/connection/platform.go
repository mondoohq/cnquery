// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

const (
	DiscoveryAll           = "all"
	DiscoveryAuto          = "auto"
	DiscoveryOrganizations = "organizations"
	DiscoveryProjects      = "projects"
)

const (
	PlatformIdNeonOrganization = "//platformid.api.mondoo.app/runtime/neon/organization/"
	PlatformIdNeonProject      = "//platformid.api.mondoo.app/runtime/neon/project/"
)

func NewNeonOrganizationPlatform(orgID string) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "neon", "organization", orgID},
	}
	PlatformByName("neon-organization").Apply(pf)
	return pf
}

func NewNeonProjectPlatform(orgID, projectID string) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "neon", "organization", orgID, "project", projectID},
	}
	PlatformByName("neon-project").Apply(pf)
	return pf
}

func NewNeonOrganizationIdentifier(orgID string) string {
	return PlatformIdNeonOrganization + orgID
}

func NewNeonProjectIdentifier(projectID string) string {
	return PlatformIdNeonProject + projectID
}
