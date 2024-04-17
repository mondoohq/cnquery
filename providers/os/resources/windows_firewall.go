// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v11/providers/os/resources/windows"
)

func (w *mqlWindowsFirewallProfile) id() (string, error) {
	return w.InstanceID.Data, nil
}

func (w *mqlWindowsFirewallRule) id() (string, error) {
	return w.InstanceID.Data, nil
}

func (w *mqlWindowsFirewall) settings() (map[string]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_SETTINGS)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwSettings, err := windows.ParseWindowsFirewallSettings(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(fwSettings)
}

func (w *mqlWindowsFirewall) profiles() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_PROFILES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwProfiles, err := windows.ParseWindowsFirewallProfiles(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert firewall profiles to MQL resource
	mqlFwProfiles := make([]interface{}, len(fwProfiles))
	for i, p := range fwProfiles {

		mqlFwProfile, err := CreateResource(w.MqlRuntime, "windows.firewall.profile", map[string]*llx.RawData{
			"instanceID":                      llx.StringData(p.InstanceID),
			"name":                            llx.StringData(p.Name),
			"enabled":                         llx.IntData(p.Enabled),
			"defaultInboundAction":            llx.IntData(p.DefaultInboundAction),
			"defaultOutboundAction":           llx.IntData(p.DefaultOutboundAction),
			"allowInboundRules":               llx.IntData(p.AllowInboundRules),
			"allowLocalFirewallRules":         llx.IntData(p.AllowLocalFirewallRules),
			"allowLocalIPsecRules":            llx.IntData(p.AllowLocalIPsecRules),
			"allowUserApps":                   llx.IntData(p.AllowUserApps),
			"allowUserPorts":                  llx.IntData(p.AllowUserPorts),
			"allowUnicastResponseToMulticast": llx.IntData(p.AllowUnicastResponseToMulticast),
			"notifyOnListen":                  llx.IntData(p.NotifyOnListen),
			"enableStealthModeForIPsec":       llx.IntData(p.EnableStealthModeForIPsec),
			"logMaxSizeKilobytes":             llx.IntData(p.LogMaxSizeKilobytes),
			"logAllowed":                      llx.IntData(p.LogAllowed),
			"logBlocked":                      llx.IntData(p.LogBlocked),
			"logIgnored":                      llx.IntData(p.LogIgnored),
			"logFileName":                     llx.StringData(p.LogFileName),
		})
		if err != nil {
			return nil, err
		}

		mqlFwProfiles[i] = mqlFwProfile
	}

	return mqlFwProfiles, nil
}

func (w *mqlWindowsFirewall) rules() ([]interface{}, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// query firewall rules
	encodedCmd := powershell.Encode(windows.FIREWALL_RULES)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwRules, err := windows.ParseWindowsFirewallRules(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert firewall rules to MQL resource
	mqlFwRules := make([]interface{}, len(fwRules))
	for i, r := range fwRules {

		mqlFwRule, err := CreateResource(w.MqlRuntime, "windows.firewall.rule", map[string]*llx.RawData{
			"instanceID":            llx.StringData(r.InstanceID),
			"name":                  llx.StringData(r.Name),
			"displayName":           llx.StringData(r.DisplayName),
			"description":           llx.StringData(r.Description),
			"displayGroup":          llx.StringData(r.DisplayGroup),
			"enabled":               llx.IntData(r.Enabled),
			"direction":             llx.IntData(r.Direction),
			"action":                llx.IntData(r.Action),
			"edgeTraversalPolicy":   llx.IntData(r.EdgeTraversalPolicy),
			"looseSourceMapping":    llx.BoolData(r.LooseSourceMapping),
			"localOnlyMapping":      llx.BoolData(r.LocalOnlyMapping),
			"primaryStatus":         llx.IntData(r.PrimaryStatus),
			"status":                llx.StringData(r.Status),
			"enforcementStatus":     llx.StringData(r.EnforcementStatus),
			"policyStoreSource":     llx.StringData(r.PolicyStoreSource),
			"policyStoreSourceType": llx.IntData(r.PolicyStoreSourceType),
		})
		if err != nil {
			return nil, err
		}

		mqlFwRules[i] = mqlFwRule
	}

	return mqlFwRules, nil
}
