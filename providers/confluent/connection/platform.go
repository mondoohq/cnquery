// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"net/url"
	"regexp"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// PlatformIdConfluentOrg prefixes the identifier of a Confluent Cloud
// organization asset. The organization is the whole scope an API key reaches,
// so it is what the asset stands for.
const PlatformIdConfluentOrg = "//platformid.api.mondoo.app/runtime/confluent/organization/"

// NewConfluentOrgPlatform describes a Confluent Cloud organization asset.
func NewConfluentOrgPlatform(orgID string) *inventory.Platform {
	pf := &inventory.Platform{
		TechnologyUrlSegments: []string{"saas", "confluent", "organization", orgID},
	}
	PlatformByName("confluent").Apply(pf)
	return pf
}

// NewConfluentOrgIdentifier builds the platform ID of an organization asset.
func NewConfluentOrgIdentifier(orgID string) string {
	return PlatformIdConfluentOrg + url.PathEscape(orgID)
}

// orgFromCRN pulls the organization ID out of a Confluent Resource Name, which
// every management API object carries as `metadata.resource_name`. It is the
// only place the organization ID appears on an object that is not the
// organization itself.
var orgFromCRN = regexp.MustCompile(`(?:^|/)organization=([^/]+)`)

// OrganizationFromCRN returns the organization ID a Confluent Resource Name is
// scoped to, or the empty string when the CRN carries no organization segment.
func OrganizationFromCRN(crn string) string {
	match := orgFromCRN.FindStringSubmatch(crn)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
