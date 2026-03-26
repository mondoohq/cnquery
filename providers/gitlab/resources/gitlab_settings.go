// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
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

	return args, nil, nil
}

func (s *mqlGitlabSettings) id() (string, error) {
	return "gitlab.settings", nil
}
