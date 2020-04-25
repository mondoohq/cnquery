package resources

import (
	"fmt"

	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (p *lumiAuditpol) id() (string, error) {
	return "auditpol", nil
}

func (p *lumiAuditpol) GetEntries() ([]interface{}, error) {
	cmd, err := p.Runtime.Motor.Transport.RunCommand("auditpol /get /category:* /r")
	if err != nil {
		return nil, fmt.Errorf("could not run auditpol")
	}

	entries, err := windows.ParseAuditpol(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	auditPolEntries := make([]interface{}, len(entries))
	for i := range entries {
		entry := entries[i]
		lumiAuditpolEntry, err := p.Runtime.CreateResource("auditpol_entry",
			"machinename", entry.MachineName,
			"policytarget", entry.PolicyTarget,
			"subcategory", entry.Subcategory,
			"subcategoryguid", entry.SubcategoryGUID,
			"inclusionsetting", entry.InclusionSetting,
			"exclusionsetting", entry.ExclusionSetting,
		)
		if err != nil {
			return nil, err
		}
		auditPolEntries[i] = lumiAuditpolEntry.(Auditpol_entry)
	}

	return auditPolEntries, nil
}

func (p *lumiAuditpol_entry) id() (string, error) {
	return "auditpol_entry", nil
}
