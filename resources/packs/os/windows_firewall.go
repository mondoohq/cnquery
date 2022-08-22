package os

import (
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
	"go.mondoo.io/mondoo/resources/packs/os/windows"
)

func (w *mqlWindowsFirewall) id() (string, error) {
	return "windows.firewall", nil
}

func (w *mqlWindowsFirewallProfile) id() (string, error) {
	return w.InstanceID()
}

func (w *mqlWindowsFirewallRule) id() (string, error) {
	return w.InstanceID()
}

func (w *mqlWindowsFirewall) GetSettings() (map[string]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_SETTINGS)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwSettings, err := windows.ParseWindowsFirewallSettings(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(fwSettings)
}

func (w *mqlWindowsFirewall) GetProfiles() ([]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_PROFILES)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
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

		mqlFwProfile, err := w.MotorRuntime.CreateResource("windows.firewall.profile",
			"instanceID", p.InstanceID,
			"name", p.Name,
			"enabled", p.Enabled,
			"logFileName", p.LogFileName,
			"defaultInboundAction", p.DefaultInboundAction,
			"defaultOutboundAction", p.DefaultOutboundAction,
			"allowInboundRules", p.AllowInboundRules,
			"allowLocalFirewallRules", p.AllowLocalFirewallRules,
			"allowLocalIPsecRules", p.AllowLocalIPsecRules,
			"allowUserApps", p.AllowUserApps,
			"allowUserPorts", p.AllowUserPorts,
			"allowUnicastResponseToMulticast", p.AllowUnicastResponseToMulticast,
			"notifyOnListen", p.NotifyOnListen,
			"enableStealthModeForIPsec", p.EnableStealthModeForIPsec,
			"logMaxSizeKilobytes", p.LogMaxSizeKilobytes,
			"logAllowed", p.LogAllowed,
			"logBlocked", p.LogBlocked,
			"logIgnored", p.LogIgnored,
			"logFileName", p.LogFileName,
		)
		if err != nil {
			return nil, err
		}

		mqlFwProfiles[i] = mqlFwProfile.(WindowsFirewallProfile)
	}

	return mqlFwProfiles, nil
}

func (w *mqlWindowsFirewall) GetRules() ([]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// query firewall rules
	encodedCmd := powershell.Encode(windows.FIREWALL_RULES)
	executedCmd, err := osProvider.RunCommand(encodedCmd)
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

		mqlFwRule, err := w.MotorRuntime.CreateResource("windows.firewall.rule",
			"instanceID", r.InstanceID,
			"name", r.Name,
			"displayName", r.DisplayName,
			"description", r.Description,
			"displayGroup", r.DisplayGroup,
			"enabled", r.Enabled,
			"direction", r.Direction,
			"action", r.Action,
			"edgeTraversalPolicy", r.EdgeTraversalPolicy,
			"looseSourceMapping", r.LooseSourceMapping,
			"localOnlyMapping", r.LocalOnlyMapping,
			"primaryStatus", r.PrimaryStatus,
			"status", r.Status,
			"enforcementStatus", r.EnforcementStatus,
			"policyStoreSource", r.PolicyStoreSource,
			"policyStoreSourceType", r.PolicyStoreSourceType,
		)
		if err != nil {
			return nil, err
		}

		mqlFwRules[i] = mqlFwRule.(WindowsFirewallRule)
	}

	return mqlFwRules, nil
}
