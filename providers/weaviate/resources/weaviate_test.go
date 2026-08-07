// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/weaviate/weaviate-go-client/v5/weaviate/fault"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/rbac"
)

func TestModuleNames(t *testing.T) {
	// A decoded module-config object yields its sorted keys.
	got := moduleNames(map[string]any{"text2vec-openai": map[string]any{}, "generative-cohere": nil})
	want := []any{"generative-cohere", "text2vec-openai"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("moduleNames = %v, want %v", got, want)
	}

	// Non-object values yield an empty list, never nil.
	if got := moduleNames(nil); got == nil || len(got) != 0 {
		t.Errorf("moduleNames(nil) = %v, want empty non-nil", got)
	}
	if got := moduleNames("not-a-map"); len(got) != 0 {
		t.Errorf("moduleNames(string) = %v, want empty", got)
	}
}

func TestBuiltinRoles(t *testing.T) {
	for _, name := range []string{"root", "admin", "viewer", "read-only"} {
		if _, ok := builtinRoles[name]; !ok {
			t.Errorf("%q should be a built-in role", name)
		}
	}
	if _, ok := builtinRoles["articleReader"]; ok {
		t.Error("a custom role should not be treated as built-in")
	}
}

func TestIsForbidden(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&fault.WeaviateClientError{StatusCode: 401}, true},
		{&fault.WeaviateClientError{StatusCode: 403}, true},
		{&fault.WeaviateClientError{StatusCode: 404}, false},
		{&fault.WeaviateClientError{StatusCode: 500}, false},
		{errors.New("plain error"), false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isForbidden(c.err); got != c.want {
			t.Errorf("isForbidden(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestFlattenPermissions(t *testing.T) {
	role := &rbac.Role{
		Name:        "articleReader",
		Data:        []rbac.DataPermission{{Actions: []string{"read_data"}, Collection: "Article"}},
		Collections: []rbac.CollectionsPermission{{Actions: []string{"read_collections"}, Collection: "Article"}},
		Roles:       []rbac.RolesPermission{{Actions: []string{"read_roles", "manage_roles"}}},
	}
	got := flattenPermissions(role)
	// One entry per action: 1 data + 1 collections + 2 roles = 4.
	if len(got) != 4 {
		t.Fatalf("flattenPermissions returned %d entries, want 4: %+v", len(got), got)
	}

	// Collection-scoped domains carry the collection; role domain does not.
	byAction := map[string]permEntry{}
	for _, p := range got {
		byAction[p.action] = p
	}
	if byAction["read_data"].collection != "Article" {
		t.Errorf("read_data collection = %q, want Article", byAction["read_data"].collection)
	}
	if byAction["manage_roles"].collection != "" {
		t.Errorf("manage_roles collection = %q, want empty", byAction["manage_roles"].collection)
	}
}

func TestIDBuilders(t *testing.T) {
	const s = "http://localhost:8080"
	cases := map[string]string{
		collectionResourceID(s, "Article"): "http://localhost:8080/collection/Article",
		roleResourceID(s, "viewer"):        "http://localhost:8080/role/viewer",
		userResourceID(s, "root-user"):     "http://localhost:8080/user/root-user",
		nodeResourceID(s, "node1"):         "http://localhost:8080/node/node1",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("id = %q, want %q", got, want)
		}
	}
}
