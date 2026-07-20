// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlOktaRoleInternal struct {
	cacheUserID        string
	cacheGroupID       string
	cacheCustomRoleID  string
	cacheResourceSetID string
}

// newMqlOktaRole maps an Okta role assignment. principalType is "user" or
// "group" and principalID is the account or group the assignment was read from,
// which becomes the assignment's principal back-reference.
func newMqlOktaRole(runtime *plugin.Runtime, role *okta.Role, principalType, principalID string) (*mqlOktaRole, error) {
	r, err := CreateResource(runtime, "okta.role", map[string]*llx.RawData{
		"id":             llx.StringData(oktaStr(role.Id)),
		"assignmentType": llx.StringData(oktaStr(role.AssignmentType)),
		"created":        llx.TimeDataPtr(role.Created),
		"lastUpdated":    llx.TimeDataPtr(role.LastUpdated),
		"label":          llx.StringData(oktaStr(role.Label)),
		"status":         llx.StringData(oktaStr(role.Status)),
		"type":           llx.StringData(oktaStr(role.Type)),
	})
	if err != nil {
		return nil, err
	}

	mqlRole := r.(*mqlOktaRole)
	switch principalType {
	case "user":
		mqlRole.cacheUserID = principalID
	case "group":
		mqlRole.cacheGroupID = principalID
	}
	mqlRole.cacheCustomRoleID, mqlRole.cacheResourceSetID = oktaRoleTypedRefs(role)
	return mqlRole, nil
}

// oktaRoleTypedRefs extracts the custom-role and resource-set ids referenced by
// a role assignment from its HAL `_links`. Both are best-effort: standard admin
// roles carry neither, and org-wide custom-role assignments carry no resource
// set. The v5 SDK maps only the `self` link into a typed field, so the other
// links are read from the untyped AdditionalProperties.
func oktaRoleTypedRefs(role *okta.Role) (customRoleID, resourceSetID string) {
	links := role.GetLinks()
	ap := links.AdditionalProperties
	if ap == nil {
		return "", ""
	}

	if h := oktaLinkHref(ap["resource-set"]); h != "" {
		resourceSetID = lastPathSegment(h)
	}

	if strings.HasPrefix(oktaStr(role.Type), "CUSTOM") {
		if h := oktaLinkHref(ap["permissions"]); h != "" {
			customRoleID = oktaRoleIdFromPermissionsHref(h)
		}
		if customRoleID == "" {
			if h := oktaLinkHref(ap["role"]); h != "" {
				customRoleID = lastPathSegment(h)
			}
		}
	}
	return customRoleID, resourceSetID
}

func (o *mqlOktaRole) customRole() (*mqlOktaCustomRole, error) {
	return resolveOktaCustomRoleRef(o.MqlRuntime, o.cacheCustomRoleID, &o.CustomRole)
}

func (o *mqlOktaRole) resourceSet() (*mqlOktaResourceSet, error) {
	if o.cacheResourceSetID == "" {
		o.ResourceSet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "okta.resourceSet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheResourceSetID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaResourceSet), nil
}

func (o *mqlOktaRole) user() (*mqlOktaUser, error) {
	return resolveOktaUserRef(o.MqlRuntime, o.cacheUserID, &o.User)
}

func (o *mqlOktaRole) group() (*mqlOktaGroup, error) {
	return resolveOktaGroupRef(o.MqlRuntime, o.cacheGroupID, &o.Group)
}
