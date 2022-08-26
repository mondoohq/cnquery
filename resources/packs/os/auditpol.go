package os

import (
	"fmt"

	"go.mondoo.com/cnquery/resources/packs/os/windows"
)

func (p *mqlAuditpol) id() (string, error) {
	return "auditpol", nil
}

func (p *mqlAuditpol) GetList() ([]interface{}, error) {
	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	cmd, err := osProvider.RunCommand("auditpol /get /category:* /r")
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
		mqlAuditpolEntry, err := p.MotorRuntime.CreateResource("auditpol.entry",
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
		auditPolEntries[i] = mqlAuditpolEntry.(AuditpolEntry)
	}

	return auditPolEntries, nil
}

func (p *mqlAuditpolEntry) id() (string, error) {
	return p.Subcategoryguid()
}
