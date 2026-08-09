// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/arista/resources/eos"
)

// resolveAcl builds the arista.eos.acl named by a reference elsewhere in the
// configuration, returning (nil, nil) when no list by that name is defined.
//
// A dangling reference is a real and common misconfiguration: an access-group
// naming a list that does not exist permits all traffic. Callers report that
// as a null rather than an error so the surrounding query still returns, and
// they must set StateIsNull on the field before returning nil.
//
// The fields set here match the ones arista.eos.acls sets, so both paths
// produce the same __id and therefore share one cached resource.
func resolveAcl(runtime *plugin.Runtime, name string) (*mqlAristaEosAcl, error) {
	if name == "" {
		return nil, nil
	}

	rc, err := fetchRunningConfig(runtime)
	if err != nil {
		return nil, err
	}
	for _, acl := range eos.ParseAccessLists(rc) {
		if acl.Name != name {
			continue
		}
		mqlAcl, err := CreateResource(runtime, "arista.eos.acl", map[string]*llx.RawData{
			"name":   llx.StringData(acl.Name),
			"family": llx.StringData(acl.Family),
			"type":   llx.StringData(acl.Type),
		})
		if err != nil {
			return nil, err
		}
		return mqlAcl.(*mqlAristaEosAcl), nil
	}

	return nil, nil
}

// =====================================================================
// arista.eos.aclBinding (list)
// =====================================================================

// id keys a binding on everything that makes it distinct: the same list can be
// applied to several targets, and to one target in both directions and both
// address families.
func (a *mqlAristaEosAclBinding) id() (string, error) {
	for _, f := range []struct{ err error }{
		{a.Target.Error}, {a.TargetName.Error}, {a.Family.Error},
		{a.Direction.Error}, {a.AclName.Error},
	} {
		if f.err != nil {
			return "", f.err
		}
	}
	return "arista.eos.aclBinding/" + a.Target.Data + "/" + a.TargetName.Data + "/" +
		a.Family.Data + "/" + a.Direction.Data + "/" + a.AclName.Data, nil
}

func (a *mqlAristaEos) aclBindings() ([]any, error) {
	rc, err := fetchRunningConfig(a.MqlRuntime)
	if err != nil {
		return nil, err
	}
	bindings := eos.ParseAclBindings(rc)

	res := make([]any, 0, len(bindings))
	for _, b := range bindings {
		mqlBinding, err := CreateResource(a.MqlRuntime, "arista.eos.aclBinding", map[string]*llx.RawData{
			"target":     llx.StringData(b.Target),
			"targetName": llx.StringData(b.TargetName),
			"direction":  llx.StringData(b.Direction),
			"family":     llx.StringData(b.Family),
			"aclName":    llx.StringData(b.AclName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func (a *mqlAristaEosAclBinding) acl() (*mqlAristaEosAcl, error) {
	if a.AclName.Error != nil {
		return nil, a.AclName.Error
	}
	mqlAcl, err := resolveAcl(a.MqlRuntime, a.AclName.Data)
	if err != nil {
		return nil, err
	}
	if mqlAcl == nil {
		a.Acl.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlAcl, nil
}
