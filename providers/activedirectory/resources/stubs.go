// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import "fmt"

// Phase 4: ADCS — stubs until implementation

func (a *mqlActivedirectory) certificateTemplates() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) certificateAuthorities() ([]interface{}, error) {
	return nil, nil
}

func (a *mqlActivedirectory) pkiObjects() ([]interface{}, error) {
	return nil, nil
}

// id() methods for Phase 3 resources

func (a *mqlActivedirectoryOu) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryGpo) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryGpoLink) id() (string, error) {
	return fmt.Sprintf("%s/%d", a.Target.Data, a.Order.Data), nil
}

func (a *mqlActivedirectoryTrust) id() (string, error) {
	return a.TargetDomain.Data, nil
}

func (a *mqlActivedirectoryDnsZone) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

// Phase 4 id() stubs

func (a *mqlActivedirectoryCertificateTemplate) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryCertificateAuthority) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectoryPkiObject) id() (string, error) {
	return a.DistinguishedName.Data, nil
}
