// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
)

// Authorization API works against any STACKIT resource by type/id. For
// the stackit provider we scope it to the configured project.
const authResourceTypeProject = "project"

func (r *mqlStackitIam) members() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Authorization()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListMembersExecute(bgctx(), authResourceTypeProject, c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	members, _ := resp.GetMembersOk()
	out := make([]any, 0, len(members))
	for i := range members {
		m := members[i]
		args := map[string]*llx.RawData{
			"subject": llx.StringData(m.GetSubject()),
			"role":    llx.StringData(m.GetRole()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.iam.member", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitIamMember) id() (string, error) {
	return "stackit.iam.member/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Subject.Data + "/" + r.Role.Data, nil
}

func (r *mqlStackitIam) roles() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Authorization()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListRolesExecute(bgctx(), authResourceTypeProject, c.ProjectID())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	roles, _ := resp.GetRolesOk()
	out := make([]any, 0, len(roles))
	for i := range roles {
		role := roles[i]
		perms := []string{}
		for _, p := range role.GetPermissions() {
			perms = append(perms, p.GetName())
		}
		args := map[string]*llx.RawData{
			"name":        llx.StringData(role.GetName()),
			"description": llx.StringData(role.GetDescription()),
			"permissions": strSliceData(perms),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.iam.role", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitIamRole) id() (string, error) {
	return "stackit.iam.role/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

// stackit.kms and stackit.iam namespaces need stable __id values too.
func (r *mqlStackitKms) id() (string, error) {
	return "stackit.kms/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitIam) id() (string, error) {
	return "stackit.iam/" + conn(r.MqlRuntime).ProjectID(), nil
}
