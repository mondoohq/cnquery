package resources

import (
	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (w *lumiWindowsFirewall) id() (string, error) {
	return "windows.firewall", nil
}

func (w *lumiWindowsFirewallProfile) id() (string, error) {
	return w.InstanceID()
}

func (w *lumiWindowsFirewallRule) id() (string, error) {
	return w.InstanceID()
}

func (w *lumiWindowsFirewall) GetSettings() (map[string]interface{}, error) {
	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_SETTINGS)
	executedCmd, err := w.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwSettings, err := windows.ParseWindowsFirewallSettings(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}
	return jsonToDict(fwSettings)
}

func (w *lumiWindowsFirewall) GetProfiles() ([]interface{}, error) {
	// query firewall profiles
	encodedCmd := powershell.Encode(windows.FIREWALL_PROFILES)
	executedCmd, err := w.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwProfiles, err := windows.ParseWindowsFirewallProfiles(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert firewall profiles to lumi resource
	lumiFwProfiles := make([]interface{}, len(fwProfiles))
	for i, p := range fwProfiles {

		lumiFwProfile, err := w.Runtime.CreateResource("windows.firewall.profile",
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

		lumiFwProfiles[i] = lumiFwProfile.(WindowsFirewallProfile)
	}

	return lumiFwProfiles, nil
}

func (w *lumiWindowsFirewall) GetRules() ([]interface{}, error) {
	// query firewall rules
	encodedCmd := powershell.Encode(windows.FIREWALL_RULES)
	executedCmd, err := w.Runtime.Motor.Transport.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	fwRules, err := windows.ParseWindowsFirewallRules(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// convert firewall rules to lumi resource
	lumiFwRules := make([]interface{}, len(fwRules))
	for i, r := range fwRules {

		lumiFwRule, err := w.Runtime.CreateResource("windows.firewall.rule",
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

		lumiFwRules[i] = lumiFwRule.(WindowsFirewallRule)
	}

	return lumiFwRules, nil
}
