// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"reflect"
	"testing"

	authorization "github.com/stackitcloud/stackit-sdk-go/services/authorization/v2api"
	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"
	"go.mondoo.com/mql/llx"
)

// TestIamRoleArgs pins the role mapping: the shipped permissions list keeps
// only names, the descriptions land in permissionDescriptions keyed by name,
// and the optional id and etag read empty rather than failing when the
// response omits them.
func TestIamRoleArgs(t *testing.T) {
	var role authorization.Role
	if err := json.Unmarshal([]byte(`{
		"name": "project.auditor",
		"description": "Read-only access",
		"id": "role-1",
		"etag": "W/\"7\"",
		"permissions": [
			{"name": "project.read", "description": "Read project metadata"},
			{"name": "iam.read", "description": "Read IAM bindings"},
			{"name": "", "description": "orphan"}
		]
	}`), &role); err != nil {
		t.Fatalf("decoding role: %v", err)
	}
	args := iamRoleArgs(&role)
	if got := args["id"].Value; got != "role-1" {
		t.Fatalf("id = %v", got)
	}
	if got := args["etag"].Value; got != `W/"7"` {
		t.Fatalf("etag = %v", got)
	}
	perms, _ := args["permissions"].Value.([]any)
	if !reflect.DeepEqual(perms, []any{"project.read", "iam.read", ""}) {
		t.Fatalf("permissions = %v", perms)
	}
	descs, _ := args["permissionDescriptions"].Value.(map[string]any)
	want := map[string]any{"project.read": "Read project metadata", "iam.read": "Read IAM bindings"}
	if !reflect.DeepEqual(descs, want) {
		t.Fatalf("permissionDescriptions = %v, want %v (unnamed entry dropped)", descs, want)
	}

	var bare authorization.Role
	if err := json.Unmarshal([]byte(`{"name": "custom", "description": "", "permissions": []}`), &bare); err != nil {
		t.Fatalf("decoding bare role: %v", err)
	}
	bareArgs := iamRoleArgs(&bare)
	if bareArgs["id"].Value != "" || bareArgs["etag"].Value != "" {
		t.Fatalf("absent id/etag = %v/%v, want empty", bareArgs["id"].Value, bareArgs["etag"].Value)
	}
	if d, _ := bareArgs["permissionDescriptions"].Value.(map[string]any); len(d) != 0 {
		t.Fatalf("empty permissions gave descriptions %v", d)
	}
}

// TestFilterIamMembers pins the reverse edges: role.members and
// serviceAccount.roleBindings are both filters over the one member list, and
// anything in that list that is not a member resource is dropped rather than
// crashing the filter.
func TestFilterIamMembers(t *testing.T) {
	mk := func(subject, role string) *mqlStackitIamMember {
		m := &mqlStackitIamMember{}
		m.Subject.Data = subject
		m.Role.Data = role
		return m
	}
	items := []any{
		mk("sa-a@sa.stackit.cloud", "owner"),
		mk("alice@example.test", "owner"),
		mk("sa-a@sa.stackit.cloud", "reader"),
		"not a member",
		nil,
	}

	owners := filterIamMembers(items, func(m *mqlStackitIamMember) bool { return m.Role.Data == "owner" })
	if len(owners) != 2 {
		t.Fatalf("owners = %d, want 2", len(owners))
	}
	saBindings := filterIamMembers(items, func(m *mqlStackitIamMember) bool { return m.Subject.Data == "sa-a@sa.stackit.cloud" })
	if len(saBindings) != 2 {
		t.Fatalf("service account bindings = %d, want 2", len(saBindings))
	}
	if none := filterIamMembers(items, func(m *mqlStackitIamMember) bool { return m.Role.Data == "editor" }); len(none) != 0 {
		t.Fatalf("editor bindings = %v, want empty", none)
	}
	if got := filterIamMembers(nil, func(*mqlStackitIamMember) bool { return true }); len(got) != 0 {
		t.Fatalf("nil list = %v, want empty", got)
	}
}

// TestProjectAncestry pins the ancestry mapping and the organization lookup:
// the organization is found by type, not by position, and a chain without
// one reads empty rather than picking the last entry.
func TestProjectAncestry(t *testing.T) {
	var resp resourcemanager.GetProjectResponse
	if err := json.Unmarshal([]byte(`{
		"projectId": "11111111-1111-1111-1111-111111111111",
		"containerId": "my-project-abc123",
		"name": "my-project",
		"lifecycleState": "ACTIVE",
		"creationTime": "2026-01-02T03:04:05Z",
		"updateTime": "2026-02-03T04:05:06Z",
		"parent": {"id": "22222222-2222-2222-2222-222222222222", "containerId": "team-folder-def456", "type": "FOLDER"},
		"parents": [
			{"id": "22222222-2222-2222-2222-222222222222", "containerId": "team-folder-def456", "name": "team", "type": "FOLDER", "containerParentId": "acme-org-ghi789", "parentId": "33333333-3333-3333-3333-333333333333"},
			{"id": "33333333-3333-3333-3333-333333333333", "containerId": "acme-org-ghi789", "name": "acme", "type": "ORGANIZATION"}
		]
	}`), &resp); err != nil {
		t.Fatalf("decoding project: %v", err)
	}
	parents := resp.GetParents()

	if got := projectOrganizationID(parents); got != "acme-org-ghi789" {
		t.Fatalf("organizationId = %q, want acme-org-ghi789", got)
	}
	folder := projectAncestorArgs(&parents[0])
	if folder["id"].Value != "team-folder-def456" || folder["type"].Value != "FOLDER" || folder["parentId"].Value != "acme-org-ghi789" {
		t.Fatalf("folder ancestor = %v", valuesOf(folder))
	}
	org := projectAncestorArgs(&parents[1])
	if org["id"].Value != "acme-org-ghi789" || org["type"].Value != "ORGANIZATION" || org["parentId"].Value != "" {
		t.Fatalf("organization ancestor = %v (parentId must be empty at the top)", valuesOf(org))
	}

	// A chain that carries only folders names no organization.
	if got := projectOrganizationID(parents[:1]); got != "" {
		t.Fatalf("folder-only chain organizationId = %q, want empty", got)
	}
	if got := projectOrganizationID(nil); got != "" {
		t.Fatalf("empty chain organizationId = %q, want empty", got)
	}
}

func valuesOf(args map[string]*llx.RawData) map[string]any {
	out := map[string]any{}
	for k, v := range args {
		out[k] = v.Value
	}
	return out
}
