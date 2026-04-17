// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"io"
	"sort"
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

		valuesT := strings.Split(rawValue, ",")
		sort.Strings(valuesT)

		values := make([]any, len(valuesT))
		for i := range valuesT {
			val := valuesT[i]
			val = strings.Replace(val, "*S", "S", 1)
			values[i] = val
		}

		res.PrivilegeRights[entry.Name()] = values
	}

	return res, nil
}

// SecpolScript exports the local security policy and resolves any non-SID
// account names in the Privilege Rights section to their SIDs so that
// checks work correctly on non-English Windows installations.
const SecpolScript = `
secedit /export /cfg out.cfg | Out-Null
$raw = Get-Content out.cfg
Remove-Item .\out.cfg | Out-Null
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
            $v = $v.Trim()
            if ($v -match '^\*S-' -or $v -eq '') {
                $rs += $v
            } else {
                try {
                    $a = [System.Security.Principal.NTAccount]::new($v)
                    $rs += ('*' + $a.Translate([System.Security.Principal.SecurityIdentifier]).Value)
                } catch {
                    $rs += $v
                }
            }
        }
        $out += ($k + ' = ' + ($rs -join ','))
    } else {
        $out += $l
    }
}
Write-Output $out
`
