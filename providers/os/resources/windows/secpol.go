// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"gopkg.in/ini.v1"
)

type Secpol struct {
	SystemAccess    map[string]any
	EventAudit      map[string]any
	RegistryValues  map[string]any
	PrivilegeRights map[string]any
}

func ParseSecpol(r io.Reader) (*Secpol, error) {
	res := &Secpol{
		SystemAccess:    map[string]any{}, // except for NewAdministratorName & NewGuestName, parse everything as int64
		EventAudit:      map[string]any{}, // parse to int
		RegistryValues:  map[string]any{}, // keep strings
		PrivilegeRights: map[string]any{}, // split entries with ,
	}

	cfg, err := ini.Load(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse secpol")
	}

	sysAccess, err := cfg.GetSection("System Access")
	if err != nil {
		return nil, err
	}
	keys := sysAccess.Keys()
	for i := range keys {
		entry := keys[i]
		key := entry.Name()
		rawValue := entry.Value()

		if key == "NewAdministratorName" || key == "NewGuestName" {
			res.SystemAccess[key] = rawValue
			continue
		}

		res.SystemAccess[key] = rawValue
	}

	eventAudit, err := cfg.GetSection("Event Audit")
	if err != nil {
		return nil, err
	}
	keys = eventAudit.Keys()
	for i := range keys {
		entry := keys[i]

		rawValue := entry.Value()
		res.EventAudit[entry.Name()] = rawValue
	}

	registryValues, err := cfg.GetSection("Registry Values")
	if err != nil {
		return nil, err
	}
	keys = registryValues.Keys()
	for i := range keys {
		entry := keys[i]
		res.RegistryValues[entry.Name()] = entry.Value()
	}

	privilegeRights, err := cfg.GetSection("Privilege Rights")
	if err != nil {
		return nil, err
	}
	keys = privilegeRights.Keys()
	for i := range keys {
		entry := keys[i]
		rawValue := entry.Value()

		rawValues := strings.Split(rawValue, ",")
		valuesT := make([]string, 0, len(rawValues))
		for i := range rawValues {
			val, ok := normalizePrivilegeRight(rawValues[i])
			if ok {
				valuesT = append(valuesT, val)
			}
		}
		sort.Strings(valuesT)

		values := make([]any, len(valuesT))
		for i := range valuesT {
			values[i] = valuesT[i]
		}

		res.PrivilegeRights[entry.Name()] = values
	}

	return res, nil
}

func normalizePrivilegeRight(value string) (string, bool) {
	sid := strings.TrimSpace(value)
	sid = strings.TrimPrefix(sid, "*")
	if !isSecurityIdentifier(sid) {
		return "", false
	}
	return sid, true
}

func isSecurityIdentifier(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) < 3 || parts[0] != "S" {
		return false
	}

	for _, part := range parts[1:] {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return false
		}
	}

	return true
}

// SecpolScript exports the local security policy and resolves any non-SID
// account names in the Privilege Rights section to their SIDs so that
// checks work correctly on non-English Windows installations.
const SecpolScript = `
secedit /export /cfg out.cfg | Out-Null
$raw = Get-Content out.cfg
Remove-Item .\out.cfg | Out-Null
function Resolve-PrivilegeRightSid($v) {
    $v = $v.Trim()
    if ($v -eq '') { return $null }
    if ($v -match '^\*S-') { return $v }
    if ($v -match '^S-') { return ('*' + $v) }
    if ($v -eq 'Guest') {
        try {
            $guest = Get-LocalUser -Name "Guest" -ErrorAction Stop
            if ($null -ne $guest.SID -and $guest.SID.Value -match '^S-') {
                return ('*' + $guest.SID.Value)
            }
        } catch {}
    }
    try {
        $a = [System.Security.Principal.NTAccount]::new($v)
        return ('*' + $a.Translate([System.Security.Principal.SecurityIdentifier]).Value)
    } catch {
        return $null
    }
}
$inPR = $false
$out = @()
foreach ($l in $raw) {
    if ($l -eq '[Privilege Rights]') { $inPR = $true; $out += $l; continue }
    if ($l -match '^\[') { $inPR = $false }
    if ($inPR -and $l -match ' = ') {
        $i = $l.IndexOf(' = ')
        $k = $l.Substring(0, $i)
        $vs = $l.Substring($i + 3) -split ','
        $rs = @()
        foreach ($v in $vs) {
            $sid = Resolve-PrivilegeRightSid $v
            if ($null -ne $sid) { $rs += $sid }
        }
        $out += ($k + ' = ' + ($rs -join ','))
    } else {
        $out += $l
    }
}
Write-Output $out
`
