// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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

	res, err := CreateResource(r.MqlRuntime, "jamf.ssoSettings", map[string]*llx.RawData{
		"__id":                          llx.StringData("jamf.ssoSettings"),
		"ssoEnabled":                    llx.BoolData(info.SsoEnabled),
		"ssoForEnrollmentEnabled":       llx.BoolData(info.SsoForEnrollmentEnabled),
		"ssoBypassAllowed":              llx.BoolData(info.SsoBypassAllowed),
		"sessionTimeout":                llx.IntData(info.SessionTimeout),
		"ssoForMacOsSelfServiceEnabled": llx.BoolData(info.SsoForMacOsSelfServiceEnabled),
		"tokenExpirationDisabled":       llx.BoolData(info.TokenExpirationDisabled),
		"userAttributeEnabled":          llx.BoolData(info.UserAttributeEnabled),
		"userAttributeName":             llx.StringData(info.UserAttributeName),
		"userMapping":                   llx.StringData(info.UserMapping),
		"enrollmentSsoForAccountDrivenEnrollmentEnabled": llx.BoolData(info.EnrollmentSsoForAccountDrivenEnrollmentEnabled),
		"idpUrl":                       llx.StringData(info.IdpUrl),
		"idpProviderType":              llx.StringData(info.IdpProviderType),
		"groupEnrollmentAccessEnabled": llx.BoolData(info.GroupEnrollmentAccessEnabled),
		"groupAttributeName":           llx.StringData(info.GroupAttributeName),
		"entityId":                     llx.StringData(info.EntityId),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlJamfSsoSettings), nil
}
