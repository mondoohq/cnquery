// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// Stub implementations for computed resource fields defined in the .lr file
// but not yet implemented. These will be replaced in future phases.

// Phase 2: Users, Groups, Computers

func (a *mqlActivedirectory) users() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) groups() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) computers() ([]interface{}, error) {
	return nil, nil
}

// Phase 3: OUs, GPOs, Trusts, DNS Zones

func (a *mqlActivedirectory) organizationalUnits() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) gpos() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) trusts() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) dnsZones() ([]interface{}, error) {
	return nil, nil
}

// Phase 4: ADCS

func (a *mqlActivedirectory) certificateTemplates() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) certificateAuthorities() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) pkiObjects() ([]interface{}, error) {
	return nil, nil
}

// Stub id() methods for stub resources

func (a *mqlActivedirectoryUser) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryGroup) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryComputer) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryOu) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryGpo) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryTrust) id() (string, error) {
	return a.TargetDomain.Data, nil
}

func (a *mqlActivedirectoryCertificateTemplate) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryCertificateAuthority) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryPkiObject) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryDnsZone) id() (string, error) {
	return a.DistinguishedName.Data, nil
}
