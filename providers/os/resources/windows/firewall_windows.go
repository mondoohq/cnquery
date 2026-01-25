// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"runtime"

	wmi "github.com/StackExchange/wmi"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// WMI structs for native Windows firewall queries
// These map to MSFT_NetFirewallProfile in root/StandardCimv2 namespace

type wmiNetFirewallProfile struct {
	Name                            string
	InstanceID                      string
	Enabled                         int
	DefaultInboundAction            int
	DefaultOutboundAction           int
	AllowInboundRules               int
	AllowLocalFirewallRules         int
	AllowLocalIPsecRules            int
	AllowUserApps                   int
	AllowUserPorts                  int
	AllowUnicastResponseToMulticast int
	NotifyOnListen                  int
	EnableStealthModeForIPsec       int
	LogMaxSizeKilobytes             int
	LogAllowed                      int
	LogBlocked                      int
	LogIgnored                      int
	LogFileName                     string
}

type wmiNetFirewallRule struct {
	Name                  string
	InstanceID            string
	DisplayName           string
	Description           string
	DisplayGroup          string
	Enabled               int
	Direction             int
	Action                int
	EdgeTraversalPolicy   int
	LooseSourceMapping    bool
	LocalOnlyMapping      bool
	PrimaryStatus         int
	Status                string
	EnforcementStatus     string
	PolicyStoreSource     string
	PolicyStoreSourceType int
}

type wmiNetFirewallSetting struct {
	Name                                    string
	InstanceID                              string
	Exemptions                              int
	EnableStatefulFtp                       int
	EnableStatefulPptp                      int
	ActiveProfile                           int
	RequireFullAuthSupport                  int
	CertValidationLevel                     int
	AllowIPsecThroughNAT                    int
	MaxSAIdleTimeSeconds                    string
	KeyEncoding                             int
	EnablePacketQueuing                     int
	ElementName                             string
	Profile                                 int
	RemoteMachineTransportAuthorizationList string
	RemoteMachineTunnelAuthorizationList    string
	RemoteUserTransportAuthorizationList    string
	RemoteUserTunnelAuthorizationList       string
}

const firewallWmiNamespace = `root\StandardCimv2`

// GetNativeFirewallProfiles retrieves firewall profiles using native Windows WMI API
func GetNativeFirewallProfiles(conn shared.Connection) ([]WindowsFirewallProfile, error) {
	if conn.Type() != shared.Type_Local || runtime.GOOS != "windows" {
		return nil, nil
	}

	var wmiProfiles []wmiNetFirewallProfile
	query := "SELECT Name, InstanceID, Enabled, DefaultInboundAction, DefaultOutboundAction, AllowInboundRules, AllowLocalFirewallRules, AllowLocalIPsecRules, AllowUserApps, AllowUserPorts, AllowUnicastResponseToMulticast, NotifyOnListen, EnableStealthModeForIPsec, LogMaxSizeKilobytes, LogAllowed, LogBlocked, LogIgnored, LogFileName FROM MSFT_NetFirewallProfile"
	if err := wmi.QueryNamespace(query, &wmiProfiles, firewallWmiNamespace); err != nil {
		return nil, err
	}

	profiles := make([]WindowsFirewallProfile, len(wmiProfiles))
	for i, p := range wmiProfiles {
		profiles[i] = WindowsFirewallProfile{
			Name:                            p.Name,
			InstanceID:                      p.InstanceID,
			Enabled:                         int64(p.Enabled),
			DefaultInboundAction:            int64(p.DefaultInboundAction),
			DefaultOutboundAction:           int64(p.DefaultOutboundAction),
			AllowInboundRules:               int64(p.AllowInboundRules),
			AllowLocalFirewallRules:         int64(p.AllowLocalFirewallRules),
			AllowLocalIPsecRules:            int64(p.AllowLocalIPsecRules),
			AllowUserApps:                   int64(p.AllowUserApps),
			AllowUserPorts:                  int64(p.AllowUserPorts),
			AllowUnicastResponseToMulticast: int64(p.AllowUnicastResponseToMulticast),
			NotifyOnListen:                  int64(p.NotifyOnListen),
			EnableStealthModeForIPsec:       int64(p.EnableStealthModeForIPsec),
			LogMaxSizeKilobytes:             int64(p.LogMaxSizeKilobytes),
			LogAllowed:                      int64(p.LogAllowed),
			LogBlocked:                      int64(p.LogBlocked),
			LogIgnored:                      int64(p.LogIgnored),
			LogFileName:                     p.LogFileName,
		}
	}

	return profiles, nil
}

// GetNativeFirewallRules retrieves firewall rules using native Windows WMI API
func GetNativeFirewallRules(conn shared.Connection) ([]WindowsFirewallRule, error) {
	if conn.Type() != shared.Type_Local || runtime.GOOS != "windows" {
		return nil, nil
	}

	var wmiRules []wmiNetFirewallRule
	query := "SELECT Name, InstanceID, DisplayName, Description, DisplayGroup, Enabled, Direction, Action, EdgeTraversalPolicy, LooseSourceMapping, LocalOnlyMapping, PrimaryStatus, Status, EnforcementStatus, PolicyStoreSource, PolicyStoreSourceType FROM MSFT_NetFirewallRule"
	if err := wmi.QueryNamespace(query, &wmiRules, firewallWmiNamespace); err != nil {
		return nil, err
	}

	rules := make([]WindowsFirewallRule, len(wmiRules))
	for i, r := range wmiRules {
		rules[i] = WindowsFirewallRule{
			Name:                  r.Name,
			InstanceID:            r.InstanceID,
			DisplayName:           r.DisplayName,
			Description:           r.Description,
			DisplayGroup:          r.DisplayGroup,
			Enabled:               int64(r.Enabled),
			Direction:             int64(r.Direction),
			Action:                int64(r.Action),
			EdgeTraversalPolicy:   int64(r.EdgeTraversalPolicy),
			LooseSourceMapping:    r.LooseSourceMapping,
			LocalOnlyMapping:      r.LocalOnlyMapping,
			PrimaryStatus:         int64(r.PrimaryStatus),
			Status:                r.Status,
			EnforcementStatus:     r.EnforcementStatus,
			PolicyStoreSource:     r.PolicyStoreSource,
			PolicyStoreSourceType: int64(r.PolicyStoreSourceType),
		}
	}

	return rules, nil
}

// GetNativeFirewallSettings retrieves firewall settings using native Windows WMI API
func GetNativeFirewallSettings(conn shared.Connection) (*WindowsFirewallSettings, error) {
	if conn.Type() != shared.Type_Local || runtime.GOOS != "windows" {
		return nil, nil
	}

	var wmiSettings []wmiNetFirewallSetting
	query := "SELECT Name, InstanceID, Exemptions, EnableStatefulFtp, EnableStatefulPptp, ActiveProfile, RequireFullAuthSupport, CertValidationLevel, AllowIPsecThroughNAT, MaxSAIdleTimeSeconds, KeyEncoding, EnablePacketQueuing, ElementName, Profile, RemoteMachineTransportAuthorizationList, RemoteMachineTunnelAuthorizationList, RemoteUserTransportAuthorizationList, RemoteUserTunnelAuthorizationList FROM MSFT_NetFirewallSetting"
	if err := wmi.QueryNamespace(query, &wmiSettings, firewallWmiNamespace); err != nil {
		return nil, err
	}

	if len(wmiSettings) == 0 {
		return &WindowsFirewallSettings{}, nil
	}

	s := wmiSettings[0]
	return &WindowsFirewallSettings{
		Name:                                    s.Name,
		InstanceID:                              s.InstanceID,
		Exemptions:                              int64(s.Exemptions),
		EnableStatefulFtp:                       int64(s.EnableStatefulFtp),
		EnableStatefulPptp:                      int64(s.EnableStatefulPptp),
		ActiveProfile:                           int64(s.ActiveProfile),
		RequireFullAuthSupport:                  int64(s.RequireFullAuthSupport),
		CertValidationLevel:                     int64(s.CertValidationLevel),
		AllowIPsecThroughNAT:                    int64(s.AllowIPsecThroughNAT),
		MaxSAIdleTimeSeconds:                    s.MaxSAIdleTimeSeconds,
		KeyEncoding:                             int64(s.KeyEncoding),
		EnablePacketQueuing:                     int64(s.EnablePacketQueuing),
		ElementName:                             s.ElementName,
		Profile:                                 int64(s.Profile),
		RemoteMachineTransportAuthorizationList: s.RemoteMachineTransportAuthorizationList,
		RemoteMachineTunnelAuthorizationList:    s.RemoteMachineTunnelAuthorizationList,
		RemoteUserTransportAuthorizationList:    s.RemoteUserTransportAuthorizationList,
		RemoteUserTunnelAuthorizationList:       s.RemoteUserTunnelAuthorizationList,
	}, nil
}
