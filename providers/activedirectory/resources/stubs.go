// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "fmt"

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

func (a *mqlActivedirectoryDangerousPermission) id() (string, error) {
	return fmt.Sprintf("%s/%s/%s", a.TargetDN.Data, a.PrincipalSID.Data, a.RightType.Data), nil
}
