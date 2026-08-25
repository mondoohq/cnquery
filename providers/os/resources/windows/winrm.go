// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// PSGetWinRMListeners enumerates the configured WS-Management listeners.
//
// The listener configuration is not in the registry with the rest of the WinRM
// policy, so it is read through the WSMan provider. Each listener is a
// container whose children are the individual settings, which is why the
// settings are collected into a hashtable rather than selected directly.
//
// $ErrorActionPreference is Stop deliberately. A host with no listener
// configured is a normal, common state and yields an empty array; a
// configuration that cannot be read at all (the WinRM service is stopped) has
// to fail loudly instead, because an empty listener list satisfies every
// assertion an audit makes about listeners.
//
// Every value is emitted as a string and converted in Go. The WSMan provider
// reports Port and Enabled as strings already, and keeping the conversion on
// this side of the wire is what makes it testable without a Windows host.
const PSGetWinRMListeners = `$ErrorActionPreference='Stop'
$res=@()
foreach($l in @(Get-ChildItem -Path WSMan:\localhost\Listener)){
$h=@{}
foreach($i in @(Get-ChildItem -Path $l.PSPath)){$h[[string]$i.Name]=[string]$i.Value}
$res+=[ordered]@{Address=[string]$h['Address'];Transport=[string]$h['Transport'];Port=[string]$h['Port'];Hostname=[string]$h['Hostname'];Enabled=[string]$h['Enabled'];CertificateThumbprint=[string]$h['CertificateThumbprint']}
}
ConvertTo-Json -InputObject @($res) -Depth 4 -Compress`

// PSGetWinRMConfig reads the WS-Management client and service settings that
// have no registry equivalent.
//
// These are the live values, which already carry any Group Policy setting: the
// WinRM service applies the policy key to its own configuration, so reading
// the policy key alone would miss a TrustedHosts set locally with
// `winrm set winrm/config/client`, which is exactly the configuration worth
// finding.
const PSGetWinRMConfig = `$ErrorActionPreference='Stop'
[ordered]@{
Client=[ordered]@{TrustedHosts=[string](Get-Item -Path WSMan:\localhost\Client\TrustedHosts).Value}
Service=[ordered]@{IPv4Filter=[string](Get-Item -Path WSMan:\localhost\Service\IPv4Filter).Value;IPv6Filter=[string](Get-Item -Path WSMan:\localhost\Service\IPv6Filter).Value}
}|ConvertTo-Json -Depth 4 -Compress`

// PSScalar decodes a WS-Management setting value.
//
// The WSMan provider reports every setting as a string, but ConvertTo-Json
// renders a value that reached it as a number or a boolean without quotes. A
// plain string tag fails the decode of the whole payload on one such value, so
// the number or boolean is kept verbatim instead. An absent value serializes
// as an empty object rather than as null, which is also flattened to empty.
type PSScalar string

func (s *PSScalar) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || data[0] == '{' || data[0] == '[' {
		*s = ""
		return nil
	}
	if data[0] == '"' {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = PSScalar(v)
		return nil
	}
	*s = PSScalar(string(data))
	return nil
}

// WinRMListener is one configured WS-Management listener.
type WinRMListener struct {
	Address               PSScalar `json:"Address"`
	Transport             PSScalar `json:"Transport"`
	Port                  PSScalar `json:"Port"`
	Hostname              PSScalar `json:"Hostname"`
	Enabled               PSScalar `json:"Enabled"`
	CertificateThumbprint PSScalar `json:"CertificateThumbprint"`
}

// PortNumber returns the listener port, or 0 when it cannot be read. Reporting
// 0 rather than guessing 5985 keeps an unreadable port distinguishable from a
// listener that really is on the default port.
func (l WinRMListener) PortNumber() int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(string(l.Port)), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// IsEnabled reports whether the listener is enabled. The WSMan provider writes
// the value as the string "true" or "false"; a value that is neither is read
// as disabled.
func (l WinRMListener) IsEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(string(l.Enabled)), "true")
}

// ID builds the resource id of a listener.
//
// A listener is addressed by its address and transport together, which is also
// what makes the pair unique: one host can carry an HTTP and an HTTPS listener
// on the same address, and two listeners on different addresses with the same
// transport. Both dimensions are in the id, or the second listener reports the
// first one's port and certificate.
func (l WinRMListener) ID() string {
	return "windows.winrm.listener/" + string(l.Address) + "+" + string(l.Transport)
}

// ParseWinRMListeners decodes the listener list.
//
// It accepts every shape PowerShell produces for a list. An empty list is a
// bare [], a normal list is an array, and a single listener can arrive as a
// bare object when PowerShell flattens the one-element array away. A list that
// passed through a calculated property arrives as {"value":[...],"Count":n},
// which a plain slice tag would decode to empty and report as "no listeners"
// on a host that has one.
func ParseWinRMListeners(r io.Reader) ([]WinRMListener, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return []WinRMListener{}, nil
	}

	if data[0] == '{' {
		// tell the Count-wrapped list apart from a single flattened listener:
		// only the wrapper carries a "value" key holding an array
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil, err
		}
		if raw, ok := probe["value"]; ok {
			if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && trimmed[0] == '[' {
				data = trimmed
			}
		}
		if data[0] == '{' {
			var single WinRMListener
			if err := json.Unmarshal(data, &single); err != nil {
				return nil, err
			}
			return []WinRMListener{single}, nil
		}
	}

	res := []WinRMListener{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// WinRMConfig is the parsed result of PSGetWinRMConfig.
type WinRMConfig struct {
	Client  WinRMClientConfig  `json:"Client"`
	Service WinRMServiceConfig `json:"Service"`
}

// WinRMClientConfig is the WS-Management client configuration.
type WinRMClientConfig struct {
	TrustedHosts PSScalar `json:"TrustedHosts"`
}

// WinRMServiceConfig is the WS-Management service configuration.
type WinRMServiceConfig struct {
	IPv4Filter PSScalar `json:"IPv4Filter"`
	IPv6Filter PSScalar `json:"IPv6Filter"`
}

// ParseWinRMConfig decodes the WS-Management client and service settings.
func ParseWinRMConfig(r io.Reader) (*WinRMConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return &WinRMConfig{}, nil
	}

	var res WinRMConfig
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
