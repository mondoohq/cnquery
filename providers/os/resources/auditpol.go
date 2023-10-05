// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/os/resources/windows"
)

func (p *mqlAuditpol) list() ([]interface{}, error) {
	o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("auditpol /get /category:* /r"),
	})
	if err != nil {
		return nil, err
	}

	cmd := o.(*mqlCommand)
	out := cmd.GetStdout()
	if out.Error != nil {
		return nil, fmt.Errorf("could not run auditpol: " + out.Error.Error())
	}

	entries, err := windows.ParseAuditpol(strings.NewReader(out.Data))
	if err != nil {
		return nil, err
	}

	auditPolEntries := make([]interface{}, len(entries))
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
