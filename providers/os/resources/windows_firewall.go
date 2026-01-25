// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v12/providers/os/resources/windows"
)

func (w *mqlWindowsFirewallProfile) id() (string, error) {
	return w.InstanceID.Data, nil
}

func (w *mqlWindowsFirewallRule) id() (string, error) {
	return w.InstanceID.Data, nil
}

func (w *mqlWindowsFirewall) settings() (map[string]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	// Try native API for local Windows connections (faster)
	fwSettings, err := windows.GetNativeFirewallSettings(conn)
	if err != nil {
		log.Debug().Err(err).Msg("native firewall settings API failed, falling back to PowerShell")
	}
	if fwSettings != nil {
		return convert.JsonToDict(fwSettings)
	}

	// Fallback to PowerShell for remote connections or non-Windows platforms
	encodedCmd := powershell.Encode(windows.FIREWALL_SETTINGS)
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	if executedCmd.ExitStatus != 0 {
		stderr, err := io.ReadAll(executedCmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("failed to retrieve computer info: " + string(stderr))
	}

	fwSettings, err = windows.ParseWindowsFirewallSettings(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(fwSettings)
}

func (w *mqlWindowsFirewall) profiles() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	var fwProfiles []windows.WindowsFirewallProfile

	// Try native API for local Windows connections (faster)
	nativeProfiles, err := windows.GetNativeFirewallProfiles(conn)
	if err != nil {
		log.Debug().Err(err).Msg("native firewall profiles API failed, falling back to PowerShell")
	}
	if nativeProfiles != nil {
		fwProfiles = nativeProfiles
	} else {
		// Fallback to PowerShell for remote connections or non-Windows platforms
		encodedCmd := powershell.Encode(windows.FIREWALL_PROFILES)
		executedCmd, err := conn.RunCommand(encodedCmd)
		if err != nil {
			return nil, err
		}

		if executedCmd.ExitStatus != 0 {
			stderr, err := io.ReadAll(executedCmd.Stderr)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("failed to retrieve firewall profiles: " + string(stderr))
		}

		fwProfiles, err = windows.ParseWindowsFirewallProfiles(executedCmd.Stdout)
		if err != nil {
			return nil, err
		}
	}

	// convert firewall profiles to MQL resource
	mqlFwProfiles := make([]any, len(fwProfiles))
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

func (w *mqlWindowsFirewall) rules() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)

	var fwRules []windows.WindowsFirewallRule

	// Try native API for local Windows connections (faster)
	nativeRules, err := windows.GetNativeFirewallRules(conn)
	if err != nil {
		log.Debug().Err(err).Msg("native firewall rules API failed, falling back to PowerShell")
	}
	if nativeRules != nil {
		fwRules = nativeRules
	} else {
		// Fallback to PowerShell for remote connections or non-Windows platforms
		encodedCmd := powershell.Encode(windows.FIREWALL_RULES)
		executedCmd, err := conn.RunCommand(encodedCmd)
		if err != nil {
			return nil, err
		}

		if executedCmd.ExitStatus != 0 {
			stderr, err := io.ReadAll(executedCmd.Stderr)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("failed to retrieve firewall rules: " + string(stderr))
		}

		fwRules, err = windows.ParseWindowsFirewallRules(executedCmd.Stdout)
		if err != nil {
			return nil, err
		}
	}

	// convert firewall rules to MQL resource
	mqlFwRules := make([]any, len(fwRules))
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
