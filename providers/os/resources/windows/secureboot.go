// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strings"
)

// PSConfirmSecureBoot reports Secure Boot state. Confirm-SecureBootUEFI throws
// on non-UEFI (legacy BIOS) systems, so it is wrapped in try/catch: a thrown
// exception means the host is not UEFI, which yields efi=false and enabled=false
// instead of an error. SetupMode is read from its UEFI variable when available.
const PSConfirmSecureBoot = `
$efi = $false
$enabled = $false
$setupMode = $false
try {
    $result = Confirm-SecureBootUEFI -ErrorAction Stop
    $efi = $true
    $enabled = [bool]$result
    try {
        $sm = Get-SecureBootUEFI -Name SetupMode -ErrorAction Stop
        if ($sm -and $sm.Bytes.Length -ge 1) { $setupMode = ($sm.Bytes[0] -eq 1) }
    } catch {
        $setupMode = $false
    }
} catch {
    $efi = $false
    $enabled = $false
}
[PSCustomObject]@{
  Efi       = $efi
  Enabled   = $enabled
  SetupMode = $setupMode
} | ConvertTo-Json
`

// SecureBootStatus is the parsed result of PSConfirmSecureBoot.
type SecureBootStatus struct {
	Efi       bool `json:"Efi"`
	Enabled   bool `json:"Enabled"`
	SetupMode bool `json:"SetupMode"`
}

// ParseSecureBoot decodes the JSON emitted by PSConfirmSecureBoot. Empty output
// is treated as a non-UEFI host rather than an error.
func ParseSecureBoot(r io.Reader) (*SecureBootStatus, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return &SecureBootStatus{}, nil
	}

	var status SecureBootStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
