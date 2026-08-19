// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// Authentication types New Relic reports for an authentication domain.
const (
	authTypeDisabled  = "DISABLED"
	authTypePassword  = "PASSWORD"
	authTypeSAMLSSO   = "SAML_SSO"
	authTypeOIDCSSO   = "OIDC_SSO"
	authTypeHerokuSSO = "HEROKU_SSO"
)

// Provisioning types New Relic reports for an authentication domain.
const (
	provisioningDisabled = "DISABLED"
	provisioningManual   = "MANUAL"
	provisioningSCIM     = "SCIM"
)

// isSSOAuthentication reports whether a domain's login method is a single
// sign-on one.
//
// The check is an allow list of the single sign-on methods rather than "not
// PASSWORD", so a login method this provider has never seen reports false. That
// direction fails an `ssoEnabled` assertion instead of passing it, which is the
// right way round: a new method is a reason to look, not a reason to assume the
// domain is protected.
func isSSOAuthentication(authenticationType string) bool {
	switch authenticationType {
	case authTypeSAMLSSO, authTypeOIDCSSO, authTypeHerokuSSO:
		return true
	default:
		return false
	}
}

// isPasswordAuthentication reports whether a domain accepts a New Relic
// username and password. Only the explicit PASSWORD method counts, so an
// unrecognized method is not reported as password login either.
func isPasswordAuthentication(authenticationType string) bool {
	return authenticationType == authTypePassword
}

// isScimProvisioning reports whether accounts in a domain are provisioned from
// an external directory over SCIM. Only the explicit SCIM value counts, so a
// domain whose provisioning method is unknown is not reported as automated.
func isScimProvisioning(provisioningType string) bool {
	return provisioningType == provisioningSCIM
}

// isOrganizationWideGrant reports whether an access grant reaches every account
// in the organization rather than one named account.
//
// It requires positive evidence: an organization ID and no account ID. A grant
// carrying neither is not claimed to be organization-wide, because reporting a
// grant of unknown scope as the broadest possible one would raise a finding on
// every account that has one.
func isOrganizationWideGrant(grant apiGrantedRole) bool {
	return grant.AccountID <= 0 && grant.OrganizationId != ""
}

// isRetentionRuleActive reports whether a retention rule is in force. New Relic
// returns deleted rules alongside live ones, and a deleted rule's retention
// window is no longer applied to its namespace.
func isRetentionRuleActive(rule apiRetentionRule) bool {
	return rule.DeletedAt.IsZero()
}

// hasPendingUpgrade reports whether a user has an open request to be moved to a
// higher tier. New Relic returns an empty object rather than null for a user
// with no request, so the presence of the field is not enough. The SDK models
// the request as a value rather than a pointer, which makes that doubly true:
// an absent request and an empty one are the same zero-valued struct, and only
// the ID separates either from a real request.
func hasPendingUpgrade(user apiUser) bool {
	return user.PendingUpgradeRequest.ID != ""
}

// requestedUserType names the tier a user has asked to be moved to, empty when
// there is no open request.
func requestedUserType(user apiUser) string {
	if !hasPendingUpgrade(user) {
		return ""
	}
	return user.PendingUpgradeRequest.RequestedUserType.DisplayName
}
