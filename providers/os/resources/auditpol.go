// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

func (p *mqlAuditpol) list() ([]any, error) {
	o, err := CreateResource(p.MqlRuntime, "powershell", map[string]*llx.RawData{
		"script": llx.StringData("[Console]::OutputEncoding = [Text.Encoding]::UTF8;auditpol /get /category:* /r"),
	})
	if err != nil {
		return nil, err
	}

	cmd := o.(*mqlPowershell)
	out := cmd.GetStdout()
	if out.Error != nil {
		return nil, fmt.Errorf("could not run auditpol: %w", out.Error)
	}
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, fmt.Errorf("could not run auditpol: %s", strings.TrimSpace(cmd.Stderr.Data))
	}

	entries, err := windows.ParseAuditpol(strings.NewReader(out.Data))
	if err != nil {
		return nil, err
	}

	auditPolEntries := make([]any, len(entries))
	for i := range entries {
		entry := entries[i]
		o, err := CreateResource(p.MqlRuntime, "auditpol.entry", map[string]*llx.RawData{
			"machinename":      llx.StringData(entry.MachineName),
			"policytarget":     llx.StringData(entry.PolicyTarget),
			"subcategory":      llx.StringData(entry.Subcategory),
			"subcategoryguid":  llx.StringData(entry.SubcategoryGUID),
			"inclusionsetting": llx.StringData(entry.InclusionSetting),
			"exclusionsetting": llx.StringData(entry.ExclusionSetting),
		})
		if err != nil {
			return nil, err
		}
		auditPolEntries[i] = o.(*mqlAuditpolEntry)
	}

	return auditPolEntries, nil
}

func (p *mqlAuditpolEntry) id() (string, error) {
	return p.Subcategoryguid.Data, nil
}

// auditpolInclusionAudits reports whether an auditpol inclusion setting audits
// the given event kind ("success" or "failure"). The setting is one of
// "Success", "Failure", "Success and Failure", or "No Auditing".
func auditpolInclusionAudits(inclusionSetting, kind string) bool {
	return strings.Contains(strings.ToLower(inclusionSetting), kind)
}

// success reports whether the inclusion setting audits success events. It is
// true for "Success" and "Success and Failure", false for "Failure" and
// "No Auditing".
func (p *mqlAuditpolEntry) success() (bool, error) {
	setting := p.GetInclusionsetting()
	if setting.Error != nil {
		return false, setting.Error
	}
	return auditpolInclusionAudits(setting.Data, "success"), nil
}

// failure reports whether the inclusion setting audits failure events. It is
// true for "Failure" and "Success and Failure", false for "Success" and
// "No Auditing".
func (p *mqlAuditpolEntry) failure() (bool, error) {
	setting := p.GetInclusionsetting()
	if setting.Error != nil {
		return false, setting.Error
	}
	return auditpolInclusionAudits(setting.Data, "failure"), nil
}
