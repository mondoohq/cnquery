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

// auditpolAuditFlags records whether an inclusion setting audits success and/or
// failure events.
type auditpolAuditFlags struct {
	success bool
	failure bool
}

// auditpolInclusionSettings maps every "Inclusion Setting" value auditpol /r can
// emit to whether it audits success and/or failure events. auditpol localizes
// this column to the OS display language, so the same setting appears under
// several spellings; keys are lowercased. Settings that audit neither event
// (e.g. "No Auditing" and its localized forms) are intentionally absent and
// resolve to the zero value via the map lookup. Supported languages: English,
// German, Dutch, Italian.
var auditpolInclusionSettings = map[string]auditpolAuditFlags{
	// English
	"success":             {success: true},
	"failure":             {failure: true},
	"success and failure": {success: true, failure: true},
	// German
	"erfolg":            {success: true},
	"fehler":            {failure: true},
	"erfolg und fehler": {success: true, failure: true},
	// Dutch
	"geslaagd":            {success: true},
	"mislukt":             {failure: true},
	"geslaagd en mislukt": {success: true, failure: true},
	// Italian
	"operazione riuscita":       {success: true},
	"errore":                    {failure: true},
	"esito positivo e negativo": {success: true, failure: true},
	// French. auditpol may render the capital "É" with or without its accent,
	// so accept both spellings of the failure forms.
	"succès":          {success: true},
	"échec":           {failure: true},
	"echec":           {failure: true},
	"succès et échec": {success: true, failure: true},
	"succès et echec": {success: true, failure: true},
}

// auditpolInclusionAudits reports whether the given (possibly localized)
// inclusion setting audits success and failure events. Unrecognized settings
// audit neither.
func auditpolInclusionAudits(inclusionSetting string) auditpolAuditFlags {
	return auditpolInclusionSettings[strings.ToLower(strings.TrimSpace(inclusionSetting))]
}

// success reports whether the inclusion setting audits success events. It is
// true for "Success" and "Success and Failure" (and their localized forms),
// false for "Failure" and "No Auditing".
func (p *mqlAuditpolEntry) success() (bool, error) {
	setting := p.GetInclusionsetting()
	if setting.Error != nil {
		return false, setting.Error
	}
	return auditpolInclusionAudits(setting.Data).success, nil
}

// failure reports whether the inclusion setting audits failure events. It is
// true for "Failure" and "Success and Failure" (and their localized forms),
// false for "Success" and "No Auditing".
func (p *mqlAuditpolEntry) failure() (bool, error) {
	setting := p.GetInclusionsetting()
	if setting.Error != nil {
		return false, setting.Error
	}
	return auditpolInclusionAudits(setting.Data).failure, nil
}
