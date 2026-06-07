// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strings"
)

// PSGetTpm collects Trusted Platform Module state. Get-Tpm provides the
// presence/readiness flags and the manufacturer version, while the Win32_Tpm
// WMI class provides the specification version. Errors are suppressed so a
// machine without a TPM yields present=false rather than failing.
const PSGetTpm = `
$ErrorActionPreference = 'SilentlyContinue'
$tpm = Get-Tpm
$spec = (Get-CimInstance -Namespace 'root\cimv2\Security\MicrosoftTpm' -ClassName Win32_Tpm).SpecVersion
[PSCustomObject]@{
  TpmPresent          = [bool]$tpm.TpmPresent
  TpmReady            = [bool]$tpm.TpmReady
  TpmEnabled          = [bool]$tpm.TpmEnabled
  TpmActivated        = [bool]$tpm.TpmActivated
  ManufacturerVersion = [string]$tpm.ManufacturerVersion
  SpecVersion         = [string]$spec
} | ConvertTo-Json
`

// TpmInfo is the parsed result of PSGetTpm.
type TpmInfo struct {
	TpmPresent          bool   `json:"TpmPresent"`
	TpmReady            bool   `json:"TpmReady"`
	TpmEnabled          bool   `json:"TpmEnabled"`
	TpmActivated        bool   `json:"TpmActivated"`
	ManufacturerVersion string `json:"ManufacturerVersion"`
	// SpecVersion is the raw Win32_Tpm value, e.g. "2.0, 0, 1.59".
	SpecVersion string `json:"SpecVersion"`
}

// MajorSpecVersion returns just the major specification version (e.g. "2.0")
// from the raw, comma-separated Win32_Tpm SpecVersion value.
func (t *TpmInfo) MajorSpecVersion() string {
	major, _, _ := strings.Cut(t.SpecVersion, ",")
	return strings.TrimSpace(major)
}

// ParseTpm decodes the JSON emitted by PSGetTpm. Empty output (no object
// produced) is treated as an absent TPM rather than an error.
func ParseTpm(r io.Reader) (*TpmInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return &TpmInfo{}, nil
	}

	var info TpmInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
