// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/weaviate/connection"
)

// newIntegrationRuntime connects to a live Weaviate server described by the
// WEAVIATE_TEST_* environment, or skips when it is not configured.
//
//	WEAVIATE_TEST_HOST    host or host:port (default port 8080)
//	WEAVIATE_TEST_SCHEME  http or https (default http)
//	WEAVIATE_TEST_KEY     API key (required)
func newIntegrationRuntime(t *testing.T) *plugin.Runtime {
	host := os.Getenv("WEAVIATE_TEST_HOST")
	key := os.Getenv("WEAVIATE_TEST_KEY")
	if host == "" || key == "" {
		t.Skip("set WEAVIATE_TEST_HOST and WEAVIATE_TEST_KEY to run weaviate integration tests")
	}

	options := map[string]string{}
	if h, p, ok := strings.Cut(host, ":"); ok {
		options["host"] = h
		options["port"] = p
		if _, err := strconv.Atoi(p); err != nil {
			t.Fatalf("invalid WEAVIATE_TEST_HOST port: %q", p)
		}
	} else {
		options["host"] = host
	}
	if scheme := os.Getenv("WEAVIATE_TEST_SCHEME"); scheme != "" {
		options["scheme"] = scheme
	}

	conf := &inventory.Config{
		Type:        "weaviate",
		Options:     options,
		Credentials: []*vault.Credential{vault.NewPasswordCredential("", key)},
	}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	conn, err := connection.NewWeaviateConnection(1, asset, conf)
	if err != nil {
		t.Fatalf("failed to build connection: %v", err)
	}
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

func mustInstance(t *testing.T, runtime *plugin.Runtime) *mqlWeaviateInstance {
	res, err := NewResource(runtime, "weaviate.instance", map[string]*llx.RawData{})
	if err != nil {
		t.Fatalf("failed to resolve weaviate.instance: %v", err)
	}
	return res.(*mqlWeaviateInstance)
}

func resolveList(t *testing.T, label string, tv *plugin.TValue[[]any]) []any {
	t.Helper()
	if tv.Error != nil {
		t.Errorf("%s errored: %v", label, tv.Error)
		return nil
	}
	return tv.Data
}

func TestIntegrationInstance(t *testing.T) {
	inst := mustInstance(t, newIntegrationRuntime(t))
	if v := inst.GetVersion(); v.Error != nil || v.Data == "" {
		t.Errorf("version = %q (err=%v)", v.Data, v.Error)
	}
	for _, tv := range []*plugin.TValue[bool]{
		inst.GetRbacEnabled(), inst.GetOidcEnabled(), inst.GetAnonymousAccessEnabled(),
	} {
		if tv.Error != nil {
			t.Errorf("instance bool field errored: %v", tv.Error)
		}
	}
}

// TestIntegrationResolveAll walks every resource and field getter and asserts
// each resolves without error against a live server.
func TestIntegrationResolveAll(t *testing.T) {
	inst := mustInstance(t, newIntegrationRuntime(t))

	for _, x := range resolveList(t, "collections", inst.GetCollections()) {
		c := x.(*mqlWeaviateCollection)
		if v := c.GetName(); v.Error != nil {
			t.Errorf("collection.name errored: %v", v.Error)
		}
	}
	resolveList(t, "nodes", inst.GetNodes())

	for _, x := range resolveList(t, "roles", inst.GetRoles()) {
		role := x.(*mqlWeaviateRole)
		resolveList(t, "role.permissions", role.GetPermissions())
		resolveList(t, "role.assignedUsers", role.GetAssignedUsers())
	}
	for _, x := range resolveList(t, "users", inst.GetUsers()) {
		u := x.(*mqlWeaviateUser)
		resolveList(t, "user.roles", u.GetRoles())
	}
}

// TestIntegrationSeededFixtures verifies values seeded by testdata/seed.sh.
// Enable it with WEAVIATE_TEST_SEEDED=1 after running the seed.
func TestIntegrationSeededFixtures(t *testing.T) {
	if os.Getenv("WEAVIATE_TEST_SEEDED") == "" {
		t.Skip("set WEAVIATE_TEST_SEEDED=1 after running testdata/seed.sh")
	}
	inst := mustInstance(t, newIntegrationRuntime(t))

	// The seeded Article collection has multi-tenancy and auto-tenant-creation on.
	var article *mqlWeaviateCollection
	for _, x := range inst.GetCollections().Data {
		if c := x.(*mqlWeaviateCollection); c.GetName().Data == "Article" {
			article = c
		}
	}
	if article == nil {
		t.Fatal("seeded Article collection not found")
	}
	if !article.GetMultiTenancyEnabled().Data {
		t.Error("Article should have multi-tenancy enabled")
	}
	if !article.GetAutoTenantCreation().Data {
		t.Error("Article should have auto-tenant-creation enabled")
	}

	// Built-in roles are flagged; the custom role is not.
	roles := map[string]*mqlWeaviateRole{}
	for _, x := range inst.GetRoles().Data {
		r := x.(*mqlWeaviateRole)
		roles[r.GetName().Data] = r
	}
	if r := roles["viewer"]; r == nil || !r.GetIsBuiltin().Data {
		t.Error("viewer should be a built-in role")
	}
	custom := roles["articleReader"]
	if custom == nil {
		t.Fatal("seeded articleReader role not found")
	}
	if custom.GetIsBuiltin().Data {
		t.Error("articleReader is a custom role, not built-in")
	}
	// Its permissions include read_data scoped to Article.
	foundReadData := false
	for _, p := range custom.GetPermissions().Data {
		perm := p.(*mqlWeaviateRolePermission)
		if perm.GetAction().Data == "read_data" && perm.GetCollection().Data == "Article" {
			foundReadData = true
		}
	}
	if !foundReadData {
		t.Error("articleReader should grant read_data on Article")
	}
	// The custom role is assigned to viewer-user, reported once.
	assigned := custom.GetAssignedUsers().Data
	if len(assigned) != 1 || assigned[0] != "viewer-user" {
		t.Errorf("articleReader assignedUsers = %v, want [viewer-user]", assigned)
	}
}
