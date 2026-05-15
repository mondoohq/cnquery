// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/deploymenttheory/go-api-sdk-jamfpro/sdk/jamfpro"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (s *mqlJamfSsoSettings) id() (string, error) {
	return "jamf.ssoSettings", nil
}

func (r *mqlJamf) sso() (*mqlJamfSsoSettings, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	info, err := client.GetSsoSettings()
	if err != nil {
		return nil, err
	}
	if info == nil {
		r.Sso = plugin.TValue[*mqlJamfSsoSettings]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	// As of go-api-sdk-jamfpro v1.47.0, the SAML-specific settings moved from
	// the flat ResourceSsoSettings struct into a nested, optional SamlSettings
	// struct. It is nil when SSO is disabled or configured for OIDC only.
	saml := jamfpro.SamlSettings{}
	if info.SamlSettings != nil {
		saml = *info.SamlSettings
	}

	res, err := CreateResource(r.MqlRuntime, "jamf.ssoSettings", map[string]*llx.RawData{
		"__id":                          llx.StringData("jamf.ssoSettings"),
		"ssoEnabled":                    llx.BoolData(info.SsoEnabled),
		"ssoForEnrollmentEnabled":       llx.BoolData(info.SsoForEnrollmentEnabled),
		"ssoBypassAllowed":              llx.BoolData(info.SsoBypassAllowed),
		"sessionTimeout":                llx.IntData(saml.SessionTimeout),
		"ssoForMacOsSelfServiceEnabled": llx.BoolData(info.SsoForMacOsSelfServiceEnabled),
		"tokenExpirationDisabled":       llx.BoolData(saml.TokenExpirationDisabled),
		"userAttributeEnabled":          llx.BoolData(saml.UserAttributeEnabled),
		"userAttributeName":             llx.StringData(saml.UserAttributeName),
		"userMapping":                   llx.StringData(saml.UserMapping),
		"enrollmentSsoForAccountDrivenEnrollmentEnabled": llx.BoolData(info.EnrollmentSsoForAccountDrivenEnrollmentEnabled),
		"idpUrl":                       llx.StringData(saml.IdpUrl),
		"idpProviderType":              llx.StringData(saml.IdpProviderType),
		"groupEnrollmentAccessEnabled": llx.BoolData(info.GroupEnrollmentAccessEnabled),
		"groupAttributeName":           llx.StringData(saml.GroupAttributeName),
		"entityId":                     llx.StringData(saml.EntityId),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlJamfSsoSettings), nil
}
