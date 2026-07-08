// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
	"go.mondoo.com/mql/v13/types"
)

// initGitlabSettings fetches instance-level application settings.
// Requires an admin-scoped token; returns a permission error otherwise.
func initGitlabSettings(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GitLabConnection)
	settings, _, err := conn.Client().Settings.GetSettings()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.IntData(settings.ID)
	args["updatedAt"] = llx.TimeDataPtr(settings.UpdatedAt)
	args["requireTwoFactorAuthentication"] = llx.BoolData(settings.RequireTwoFactorAuthentication)
	args["twoFactorGracePeriod"] = llx.IntData(settings.TwoFactorGracePeriod)
	args["requireAdminTwoFactorAuthentication"] = llx.BoolData(settings.RequireAdminTwoFactorAuthentication)
	args["gitTwoFactorSessionExpiry"] = llx.IntData(settings.GitTwoFactorSessionExpiry)
	args["passwordAuthenticationEnabledForWeb"] = llx.BoolData(settings.PasswordAuthenticationEnabledForWeb)
	args["passwordAuthenticationEnabledForGit"] = llx.BoolData(settings.PasswordAuthenticationEnabledForGit)
	args["signupEnabled"] = llx.BoolData(settings.SignupEnabled)
	args["defaultProjectVisibility"] = llx.StringData(string(settings.DefaultProjectVisibility))
	args["defaultGroupVisibility"] = llx.StringData(string(settings.DefaultGroupVisibility))
	args["minimumPasswordLength"] = llx.IntData(settings.MinimumPasswordLength)
	args["passwordNumberRequired"] = llx.BoolData(settings.PasswordNumberRequired)
	args["passwordSymbolRequired"] = llx.BoolData(settings.PasswordSymbolRequired)
	args["passwordUppercaseRequired"] = llx.BoolData(settings.PasswordUppercaseRequired)
	args["passwordLowercaseRequired"] = llx.BoolData(settings.PasswordLowercaseRequired)
	args["enforcePatExpiration"] = llx.BoolData(settings.EnforcePATExpiration)
	args["enforceSshKeyExpiration"] = llx.BoolData(settings.EnforceSSHKeyExpiration)
	args["requireAdminApprovalAfterUserSignup"] = llx.BoolData(settings.RequireAdminApprovalAfterUserSignup)
	args["domainAllowlist"] = llx.ArrayData(convert.SliceAnyToInterface(settings.DomainAllowlist), types.String)
	args["domainDenylistEnabled"] = llx.BoolData(settings.DomainDenylistEnabled)
	args["domainDenylist"] = llx.ArrayData(convert.SliceAnyToInterface(settings.DomainDenylist), types.String)
	args["disabledOauthSignInSources"] = llx.ArrayData(convert.SliceAnyToInterface(settings.DisabledOauthSignInSources), types.String)
	args["notifyOnUnknownSignIn"] = llx.BoolData(settings.NotifyOnUnknownSignIn)
	args["externalAuthorizationServiceEnabled"] = llx.BoolData(settings.ExternalAuthorizationServiceEnabled)
	args["allowLocalRequestsFromWebHooksAndServices"] = llx.BoolData(settings.AllowLocalRequestsFromWebHooksAndServices)
	args["allowLocalRequestsFromSystemHooks"] = llx.BoolData(settings.AllowLocalRequestsFromSystemHooks)
	args["protectedCiVariables"] = llx.BoolData(settings.ProtectedCIVariables)
	args["importSources"] = llx.ArrayData(convert.SliceAnyToInterface(settings.ImportSources), types.String)
	args["sessionExpireDelay"] = llx.IntData(settings.SessionExpireDelay)
	args["terminalMaxSessionTime"] = llx.IntData(settings.TerminalMaxSessionTime)
	args["duoFeaturesEnabled"] = llx.BoolData(settings.DuoFeaturesEnabled)
	args["lockDuoFeaturesEnabled"] = llx.BoolData(settings.LockDuoFeaturesEnabled)

	return args, nil, nil
}

func (s *mqlGitlabSettings) id() (string, error) {
	return "gitlab.settings", nil
}
