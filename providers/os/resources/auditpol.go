// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/windows"
)

func (p *mqlAuditpol) list() ([]any, error) {
	conn := p.MqlRuntime.Connection.(shared.Connection)

	var entries []windows.AuditpolEntry
	var err error

	// If running locally on Windows, use native API for better performance
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		log.Debug().Msg("using native Windows API for auditpol")
		entries, err = windows.GetNativeAuditpol()
		if err != nil {
			log.Debug().Err(err).Msg("native auditpol failed, falling back to PowerShell")
			// Fall through to PowerShell fallback
			entries = nil
		}
	}

	// Fallback to PowerShell for remote connections or if native API failed
	if entries == nil {
		entries, err = p.getAuditpolViaPowershell()
		if err != nil {
			return nil, err
		}
	}

	return p.entriesToResources(entries)
}

// getAuditpolViaPowershell retrieves audit policy using PowerShell
func (p *mqlAuditpol) getAuditpolViaPowershell() ([]windows.AuditpolEntry, error) {
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

	return windows.ParseAuditpol(strings.NewReader(out.Data))
}

// entriesToResources converts AuditpolEntry slice to MQL resources
func (p *mqlAuditpol) entriesToResources(entries []windows.AuditpolEntry) ([]any, error) {
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
