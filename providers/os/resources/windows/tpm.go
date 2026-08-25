// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strings"
)

// PSGetTpm collects Trusted Platform Module state. Get-Tpm provides the
// presence/readiness flags, the manufacturer, the dictionary-attack lockout
// state and the automatic provisioning mode, while the Win32_Tpm WMI class
// provides the specification version. Errors are suppressed so a machine
// without a TPM yields present=false rather than failing.
//
// The owner authorization value that Get-Tpm also carries is deliberately not
// collected. It is the secret that authorizes clearing and taking ownership of
// the TPM, and nothing here needs it.
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
  ManufacturerId      = $(if($null -eq $tpm.ManufacturerId){0}else{[int64]$tpm.ManufacturerId})
  ManufacturerIdTxt   = [string]$tpm.ManufacturerIdTxt
  LockedOut           = [bool]$tpm.LockedOut
  LockoutCount        = $(if($null -eq $tpm.LockoutCount){0}else{[int64]$tpm.LockoutCount})
  LockoutHealTime     = [string]$tpm.LockoutHealTime
  AutoProvisioning    = [string]$tpm.AutoProvisioning
  OwnerClearDisabled  = [bool]$tpm.OwnerClearDisabled
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
	// ManufacturerId is the raw TCG vendor identifier, usually four ASCII
	// characters packed into an integer.
	ManufacturerId int64 `json:"ManufacturerId"`
	// ManufacturerIdTxt is the readable form of the vendor identifier, which
	// the firmware pads to a fixed width, so it is trimmed on read.
	ManufacturerIdTxt string `json:"ManufacturerIdTxt"`
	// LockedOut is true while the TPM is refusing owner authorization after
	// too many failed attempts.
	LockedOut bool `json:"LockedOut"`
	// LockoutCount is the number of authorization failures counted against
	// the lockout threshold.
	LockoutCount int64 `json:"LockoutCount"`
	// LockoutHealTime is how long it takes the TPM to forget one failure,
	// reported as a human-readable duration such as "2 hours".
	LockoutHealTime string `json:"LockoutHealTime"`
	// AutoProvisioning is Enabled, Disabled, or EnabledForCertainOnly.
	AutoProvisioning string `json:"AutoProvisioning"`
	// OwnerClearDisabled is true when the TPM cannot be cleared from the
	// operating system.
	OwnerClearDisabled bool `json:"OwnerClearDisabled"`
	// SpecVersion is the raw Win32_Tpm value, e.g. "2.0, 0, 1.59".
	SpecVersion string `json:"SpecVersion"`
}

// Manufacturer returns the readable vendor identifier with the padding the
// firmware adds removed. Get-Tpm renders the fixed-width field verbatim, so an
// untrimmed value compares unequal to the vendor name it obviously is.
func (t *TpmInfo) Manufacturer() string {
	return strings.TrimSpace(strings.TrimRight(t.ManufacturerIdTxt, "\x00"))
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
