package resources

import "errors"

func (p *lumiAuditpol) id() (string, error) {
	return "auditpol", nil
}

func (p *lumiAuditpol) GetEntries() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiAuditpol_entry) id() (string, error) {
	return "auditpol_entry", nil
}
