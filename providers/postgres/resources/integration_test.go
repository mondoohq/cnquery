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
	"go.mondoo.com/mql/v13/providers/postgres/connection"
)

// newIntegrationRuntime connects to a live PostgreSQL server described by the
// PG_TEST_* environment, or skips when it is not configured.
//
//	PG_TEST_HOST      host or host:port (default port 5432)
//	PG_TEST_USER      role name (default "postgres")
//	PG_TEST_PASSWORD  password (required)
func newIntegrationRuntime(t *testing.T) *plugin.Runtime {
	host := os.Getenv("PG_TEST_HOST")
	password := os.Getenv("PG_TEST_PASSWORD")
	if host == "" || password == "" {
		t.Skip("set PG_TEST_HOST and PG_TEST_PASSWORD to run postgres integration tests")
	}
	user := os.Getenv("PG_TEST_USER")
	if user == "" {
		user = "postgres"
	}

	options := map[string]string{"sslmode": "disable"}
	if h, p, ok := strings.Cut(host, ":"); ok {
		options["host"] = h
		options["port"] = p
		if _, err := strconv.Atoi(p); err != nil {
			t.Fatalf("invalid PG_TEST_HOST port: %q", p)
		}
	} else {
		options["host"] = host
	}

	conf := &inventory.Config{
		Type:        "postgres",
		Options:     options,
		Credentials: []*vault.Credential{vault.NewPasswordCredential(user, password)},
	}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	conn, err := connection.NewPostgresConnection(1, asset, conf)
	if err != nil {
		t.Fatalf("failed to build connection: %v", err)
	}
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

func mustInstance(t *testing.T, runtime *plugin.Runtime) *mqlPostgresInstance {
	res, err := NewResource(runtime, "postgres.instance", map[string]*llx.RawData{})
	if err != nil {
		t.Fatalf("failed to resolve postgres.instance: %v", err)
	}
	return res.(*mqlPostgresInstance)
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
	if v := inst.GetVersion(); v.Error != nil || !strings.Contains(v.Data, "PostgreSQL") {
		t.Errorf("version = %q (err=%v)", v.Data, v.Error)
	}
	if v := inst.GetSystemIdentifier(); v.Error != nil || v.Data == "" {
		t.Errorf("systemIdentifier empty (err=%v)", v.Error)
	}
	if v := inst.GetPasswordEncryption(); v.Error != nil || v.Data == "" {
		t.Errorf("passwordEncryption = %q (err=%v)", v.Data, v.Error)
	}
}

// TestIntegrationResolveAll walks every resource and field getter and asserts
// each resolves without error against a live server, independent of seeded data.
func TestIntegrationResolveAll(t *testing.T) {
	inst := mustInstance(t, newIntegrationRuntime(t))

	for _, tv := range []*plugin.TValue[bool]{inst.GetSsl(), inst.GetInRecovery()} {
		if tv.Error != nil {
			t.Errorf("instance bool field errored: %v", tv.Error)
		}
	}
	resolveList(t, "settings", inst.GetSettings())
	resolveList(t, "hbaRules", inst.GetHbaRules())
	resolveList(t, "replicationSlots", inst.GetReplicationSlots())
	resolveList(t, "subscriptions", inst.GetSubscriptions())

	for _, x := range resolveList(t, "roles", inst.GetRoles()) {
		role := x.(*mqlPostgresRole)
		resolveList(t, "role.memberOf", role.GetMemberOf())
		resolveList(t, "role.members", role.GetMembers())
	}
	for _, x := range resolveList(t, "tablespaces", inst.GetTablespaces()) {
		ts := x.(*mqlPostgresTablespace)
		if ts.GetOwner().Error != nil {
			t.Errorf("tablespace.owner errored: %v", ts.GetOwner().Error)
		}
		resolveList(t, "tablespace.privileges", ts.GetPrivileges())
	}
	for _, x := range resolveList(t, "databases", inst.GetDatabases()) {
		db := x.(*mqlPostgresDatabase)
		name := db.GetName().Data
		if db.GetOwner().Error != nil {
			t.Errorf("%s.owner errored: %v", name, db.GetOwner().Error)
		}
		resolveList(t, name+".privileges", db.GetPrivileges())
		resolveList(t, name+".extensions", db.GetExtensions())
		resolveList(t, name+".publications", db.GetPublications())
		for _, f := range resolveList(t, name+".foreignServers", db.GetForeignServers()) {
			fs := f.(*mqlPostgresForeignServer)
			if fs.GetOwner().Error != nil {
				t.Errorf("%s foreignServer owner errored: %v", name, fs.GetOwner().Error)
			}
			resolveList(t, name+".foreignServer.userMappings", fs.GetUserMappings())
		}
		for _, s := range resolveList(t, name+".schemas", db.GetSchemas()) {
			sc := s.(*mqlPostgresSchema)
			if sc.GetOwner().Error != nil {
				t.Errorf("%s schema owner errored: %v", name, sc.GetOwner().Error)
			}
			resolveList(t, name+".schema.privileges", sc.GetPrivileges())
		}
		for _, f := range resolveList(t, name+".functions", db.GetFunctions()) {
			fn := f.(*mqlPostgresFunction)
			if fn.GetOwner().Error != nil {
				t.Errorf("%s function owner errored: %v", name, fn.GetOwner().Error)
			}
			resolveList(t, name+".function.privileges", fn.GetPrivileges())
		}
	}
}

// TestIntegrationSeededFixtures verifies specific values from testdata/seed.sql.
// Enable it with PG_TEST_SEEDED=1 after loading the seed.
func TestIntegrationSeededFixtures(t *testing.T) {
	if os.Getenv("PG_TEST_SEEDED") == "" {
		t.Skip("set PG_TEST_SEEDED=1 after loading testdata/seed.sql")
	}
	inst := mustInstance(t, newIntegrationRuntime(t))

	// role attributes
	var admin *mqlPostgresRole
	for _, x := range inst.GetRoles().Data {
		if r := x.(*mqlPostgresRole); r.GetName().Data == "app_admin" {
			admin = r
		}
	}
	if admin == nil {
		t.Fatal("app_admin role not found")
	}
	if !admin.GetCreateRole().Data {
		t.Error("app_admin should have createRole")
	}
	if admin.GetConnectionLimit().Data != 5 {
		t.Errorf("app_admin connectionLimit = %d, want 5", admin.GetConnectionLimit().Data)
	}
	memberOf := map[string]bool{}
	for _, x := range admin.GetMemberOf().Data {
		memberOf[x.(*mqlPostgresRole).GetName().Data] = true
	}
	if !memberOf["app_group"] {
		t.Error("app_admin should be a member of app_group")
	}

	// appdb: owner, SECURITY DEFINER function with an EXECUTE grant
	var appdb *mqlPostgresDatabase
	for _, x := range inst.GetDatabases().Data {
		if d := x.(*mqlPostgresDatabase); d.GetName().Data == "appdb" {
			appdb = d
		}
	}
	if appdb == nil {
		t.Fatal("appdb not found")
	}
	if o := appdb.GetOwner(); o.Error != nil || o.Data == nil || o.Data.GetName().Data != "app_admin" {
		t.Errorf("appdb owner not app_admin")
	}
	var secdef *mqlPostgresFunction
	for _, x := range appdb.GetFunctions().Data {
		if f := x.(*mqlPostgresFunction); f.GetName().Data == "secdef_fn" {
			secdef = f
		}
	}
	if secdef == nil {
		t.Fatal("secdef_fn not found")
	}
	if !secdef.GetIsSecurityDefiner().Data {
		t.Error("secdef_fn should be SECURITY DEFINER")
	}
	grantees := map[string]bool{}
	for _, x := range secdef.GetPrivileges().Data {
		grantees[x.(*mqlPostgresPrivilege).GetGrantee().Data] = true
	}
	if !grantees["app_group"] {
		t.Error("secdef_fn should grant EXECUTE to app_group")
	}

	// foreign server user mapping must not expose the password option
	var fs *mqlPostgresForeignServer
	for _, x := range appdb.GetForeignServers().Data {
		if s := x.(*mqlPostgresForeignServer); s.GetName().Data == "remote_srv" {
			fs = s
		}
	}
	if fs == nil {
		t.Fatal("remote_srv not found")
	}
	for _, x := range fs.GetUserMappings().Data {
		for _, opt := range x.(*mqlPostgresUserMapping).GetOptions().Data {
			if strings.HasPrefix(strings.ToLower(opt.(string)), "password") {
				t.Errorf("user mapping leaked a password option: %v", opt)
			}
		}
	}
}
