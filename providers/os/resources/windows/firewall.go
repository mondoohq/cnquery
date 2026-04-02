// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	FIREWALL_PROFILES = "Get-NetFirewallProfile | ConvertTo-Json"
	FIREWALL_RULES    = "Get-NetFirewallRule | ConvertTo-Json"
	FIREWALL_SETTINGS = "Get-NetFirewallSetting | ConvertTo-Json"
)

type WindowsFirewallRule struct {
	ID                      string  `json:"ID"`
	Name                    string  `json:"Name"`
	DisplayName             string  `json:"DisplayName"`
	Group                   string  `json:"Group"`
	Enabled                 int64   `json:"Enabled"`
	Profile                 int64   `json:"Profile"`
	Direction               int64   `json:"Direction"`
	Action                  int64   `json:"Action"`
	EdgeTraversalPolicy     int64   `json:"EdgeTraversalPolicy"`
	LSM                     bool    `json:"LSM"`
	PrimaryStatus           int64   `json:"PrimaryStatus"`
	Status                  string  `json:"Status"`
	EnforcementStatus       string  `json:"EnforcementStatus"`
	PolicyStoreSourceType   int64   `json:"PolicyStoreSourceType"`
	Caption                 *string `json:"Caption"`
	Description             string  `json:"Description"`
	ElementName             string  `json:"ElementName"`
	InstanceID              string  `json:"InstanceID"`
	PolicyDecisionStrategy  int64   `json:"PolicyDecisionStrategy"`
	ConditionListType       int64   `json:"ConditionListType"`
	CreationClassName       string  `json:"CreationClassName"`
	ExecutionStrategy       int64   `json:"ExecutionStrategy"`
	PolicyRuleName          string  `json:"PolicyRuleName"`
	SequencedActions        int64   `json:"SequencedActions"`
	SystemCreationClassName string  `json:"SystemCreationClassName"`
	SystemName              string  `json:"SystemName"`
	DisplayGroup            string  `json:"DisplayGroup"`
	LocalOnlyMapping        bool    `json:"LocalOnlyMapping"`
	LooseSourceMapping      bool    `json:"LooseSourceMapping"`
	PolicyStoreSource       string  `json:"PolicyStoreSource"`
	Profiles                int64   `json:"Profiles"`
	RuleGroup               string  `json:"RuleGroup"`
	StatusCode              int64   `json:"StatusCode"`
	// Platform    dict     `json:"Platform"`
	// CommonName            string  `json:"CommonName"`
	// PolicyKeywords            string  `json:"PolicyKeywords"`
	// PolicyRoles            string  `json:"PolicyRoles"`
	// Mandatory int64 `json:"Mandatory"`
	// Priority string `json:"Priority"`
	// RuleUsage string `json:"RuleUsage"`
	// Owner          string `json:"Owner"`
}

// streamDecodeJSONArray stream-decodes a JSON array from input, returning
// elements one at a time to avoid buffering the entire payload. It also
// handles the PowerShell quirk where a single-element result is emitted as
// a bare object instead of a one-element array.
func streamDecodeJSONArray[T any](input io.Reader) ([]T, error) {
	dec := json.NewDecoder(input)

	// Read the opening token of the JSON value.
	tok, err := dec.Token()
	if err == io.EOF {
		return []T{}, nil
	}
	if err != nil {
		return nil, err
	}

	delim, isDelim := tok.(json.Delim)

	// PowerShell emits a bare object when there is exactly one element.
	if isDelim && delim == '{' {
		var item T
		if err := json.NewDecoder(io.MultiReader(
			bytes.NewReader([]byte{'{'}),
			dec.Buffered(),
			input,
		)).Decode(&item); err != nil {
			return nil, err
		}
		return []T{item}, nil
	}

	if !isDelim || delim != '[' {
		return nil, fmt.Errorf("unexpected JSON token %v; expected '[' or '{'", tok)
	}

	// Stream-decode array elements one at a time.
	var items []T
	for dec.More() {
		var item T
		if err := dec.Decode(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func ParseWindowsFirewallRules(input io.Reader) ([]WindowsFirewallRule, error) {
	return streamDecodeJSONArray[WindowsFirewallRule](input)
}

type WindowsFirewallSettings struct {
	Name                                    string `json:"Name"`
	Exemptions                              int64  `json:"Exemptions"`
	EnableStatefulFtp                       int64  `json:"EnableStatefulFtp"`
	EnableStatefulPptp                      int64  `json:"EnableStatefulPptp"`
	ActiveProfile                           int64  `json:"ActiveProfile"`
	RequireFullAuthSupport                  int64  `json:"RequireFullAuthSupport"`
	CertValidationLevel                     int64  `json:"CertValidationLevel"`
	AllowIPsecThroughNAT                    int64  `json:"AllowIPsecThroughNAT"`
	MaxSAIdleTimeSeconds                    string `json:"MaxSAIdleTimeSeconds"`
	KeyEncoding                             int64  `json:"KeyEncoding"`
	EnablePacketQueuing                     int64  `json:"EnablePacketQueuing"`
	ElementName                             string `json:"ElementName"`
	InstanceID                              string `json:"InstanceID"`
	Profile                                 int64  `json:"Profile"`
	RemoteMachineTransportAuthorizationList string `json:"RemoteMachineTransportAuthorizationList"`
	RemoteMachineTunnelAuthorizationList    string `json:"RemoteMachineTunnelAuthorizationList"`
	RemoteUserTransportAuthorizationList    string `json:"RemoteUserTransportAuthorizationList"`
	RemoteUserTunnelAuthorizationList       string `json:"RemoteUserTunnelAuthorizationList"`
}

func ParseWindowsFirewallSettings(input io.Reader) (*WindowsFirewallSettings, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// for empty result set do not get the '{}', therefore lets abort here
	if len(data) == 0 {
		return &WindowsFirewallSettings{}, nil
	}

	var winFirewallSettings WindowsFirewallSettings
	err = json.Unmarshal(data, &winFirewallSettings)
	if err != nil {
		return nil, err
	}

	return &winFirewallSettings, nil
}

type WindowsFirewallProfile struct {
	Profile                         string  `json:"Profile"`
	Enabled                         int64   `json:"Enabled"`
	DefaultInboundAction            int64   `json:"DefaultInboundAction"`
	DefaultOutboundAction           int64   `json:"DefaultOutboundAction"`
	AllowInboundRules               int64   `json:"AllowInboundRules"`
	AllowLocalFirewallRules         int64   `json:"AllowLocalFirewallRules"`
	AllowLocalIPsecRules            int64   `json:"AllowLocalIPsecRules"`
	AllowUserApps                   int64   `json:"AllowUserApps"`
	AllowUserPorts                  int64   `json:"AllowUserPorts"`
	AllowUnicastResponseToMulticast int64   `json:"AllowUnicastResponseToMulticast"`
	NotifyOnListen                  int64   `json:"NotifyOnListen"`
	EnableStealthModeForIPsec       int64   `json:"EnableStealthModeForIPsec"`
	LogMaxSizeKilobytes             int64   `json:"LogMaxSizeKilobytes"`
	LogAllowed                      int64   `json:"LogAllowed"`
	LogBlocked                      int64   `json:"LogBlocked"`
	LogIgnored                      int64   `json:"LogIgnored"`
	Caption                         *string `json:"Caption"`
	Description                     *string `json:"Description"`
	InstanceID                      string  `json:"InstanceID"`
	LogFileName                     string  `json:"LogFileName"`
	Name                            string  `json:"Name"`
}

func ParseWindowsFirewallProfiles(input io.Reader) ([]WindowsFirewallProfile, error) {
	return streamDecodeJSONArray[WindowsFirewallProfile](input)
}
