// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ansible/connection"
)

func newProjectRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()
	asset := &inventory.Asset{Connections: []*inventory.Config{{
		Options: map[string]string{"path": "./testdata/project"},
	}}}
	conn, err := connection.NewAnsibleConnection(1, asset, &inventory.Config{})
	require.NoError(t, err)
	require.True(t, conn.IsProject(), "testdata/project must connect as a project")
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

func findRole(t *testing.T, roles []any, name string) *mqlAnsibleRole {
	t.Helper()
	for _, r := range roles {
		role := r.(*mqlAnsibleRole)
		if role.Name.Data == name {
			return role
		}
	}
	t.Fatalf("role %q not found", name)
	return nil
}

func TestProjectPlaybooks(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	playbooks, err := proj.playbooks()
	require.NoError(t, err)
	require.Len(t, playbooks, 1)

	pb := playbooks[0].(*mqlAnsiblePlaybook)
	plays, err := pb.plays()
	require.NoError(t, err)
	require.Len(t, plays, 1)
	assert.Equal(t, "Configure web servers", plays[0].(*mqlAnsiblePlay).Name.Data)
}

func TestProjectRolesAndDependencies(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	roles, err := proj.roles()
	require.NoError(t, err)
	require.Len(t, roles, 2)

	nginx := findRole(t, roles, "nginx")

	tasks, err := nginx.tasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	handlers, err := nginx.handlers()
	require.NoError(t, err)
	require.Len(t, handlers, 1)
	assert.Equal(t, "restart nginx", handlers[0].(*mqlAnsibleHandler).Name.Data)

	assert.Equal(t, 80, nginx.Defaults.Data["nginx_port"])
	assert.Equal(t, "nginx", nginx.Vars.Data["nginx_package_name"])
	assert.Contains(t, nginx.Templates.Data, "nginx.conf.j2")
	assert.Contains(t, nginx.Files.Data, "index.html")

	// meta resolves through the ansible.role.meta resource despite sharing the
	// field path with the resource name.
	meta, err := nginx.meta()
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "2.10", meta.MinAnsibleVersion.Data)

	// dependencies resolve by name to the typed common role.
	deps, err := nginx.dependencies()
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "common", deps[0].(*mqlAnsibleRole).Name.Data)
}

func TestProjectPlayRoleRefs(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	playbooks, err := proj.playbooks()
	require.NoError(t, err)
	plays, err := playbooks[0].(*mqlAnsiblePlaybook).plays()
	require.NoError(t, err)

	refs, err := plays[0].(*mqlAnsiblePlay).roleRefs()
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "nginx", refs[0].(*mqlAnsibleRole).Name.Data)
}

func TestProjectImportedTasks(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	playbooks, err := proj.playbooks()
	require.NoError(t, err)
	plays, err := playbooks[0].(*mqlAnsiblePlaybook).plays()
	require.NoError(t, err)

	pre, err := plays[0].(*mqlAnsiblePlay).preTasks()
	require.NoError(t, err)
	require.Len(t, pre, 1)

	imported, err := pre[0].(*mqlAnsibleTask).importedTasks()
	require.NoError(t, err)
	require.Len(t, imported, 2, "tasks/setup.yml has two tasks")
	assert.Equal(t, "Install base packages", imported[0].(*mqlAnsibleTask).Name.Data)
}

func TestProjectInventory(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	inv, err := proj.inventory()
	require.NoError(t, err)
	require.NotNil(t, inv)

	groups, err := inv.groups()
	require.NoError(t, err)

	webservers := findGroup(t, groups, "webservers")
	assert.ElementsMatch(t, []any{"web1.example.com", "web2.example.com"}, webservers.Hosts.Data)
	assert.Equal(t, "80", webservers.Vars.Data["http_port"])

	production := findGroup(t, groups, "production")
	assert.Contains(t, production.Children.Data, "webservers")

	all := findGroup(t, groups, "all")
	assert.Equal(t, "pool.ntp.org", all.Vars.Data["ntp_server"])

	hosts, err := inv.hosts()
	require.NoError(t, err)
	web1 := findHost(t, hosts, "web1.example.com")
	assert.Equal(t, "10.0.0.1", web1.Vars.Data["ansible_host"])
	assert.Equal(t, 1, web1.Vars.Data["server_id"])
	assert.Contains(t, web1.Groups.Data, "webservers")
}

func findGroup(t *testing.T, groups []any, name string) *mqlAnsibleInventoryGroup {
	t.Helper()
	for _, g := range groups {
		group := g.(*mqlAnsibleInventoryGroup)
		if group.Name.Data == name {
			return group
		}
	}
	t.Fatalf("group %q not found", name)
	return nil
}

func findHost(t *testing.T, hosts []any, name string) *mqlAnsibleInventoryHost {
	t.Helper()
	for _, h := range hosts {
		host := h.(*mqlAnsibleInventoryHost)
		if host.Name.Data == name {
			return host
		}
	}
	t.Fatalf("host %q not found", name)
	return nil
}

func TestProjectRequirements(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	req, err := proj.requirements()
	require.NoError(t, err)
	require.NotNil(t, req)

	roles, err := req.roles()
	require.NoError(t, err)
	require.Len(t, roles, 2)
	assert.Equal(t, "geerlingguy.nginx", roles[0].(*mqlAnsibleGalaxyRole).Name.Data)
	assert.Equal(t, "3.1.4", roles[0].(*mqlAnsibleGalaxyRole).Version.Data)

	collections, err := req.collections()
	require.NoError(t, err)
	require.Len(t, collections, 2)
	assert.Equal(t, "community.general", collections[0].(*mqlAnsibleGalaxyCollection).Name.Data)
}

func TestProjectConfig(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	cfg, err := proj.config()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.HostKeyChecking.Data, "host_key_checking = False")
	assert.True(t, cfg.Become.Data)
	assert.Equal(t, "root", cfg.BecomeUser.Data)

	defaults, ok := cfg.Sections.Data["defaults"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "./roles", defaults["roles_path"])
}

func TestProjectVault(t *testing.T) {
	rt := newProjectRuntime(t)
	proj := &mqlAnsibleProject{MqlRuntime: rt}

	vault, err := proj.vault()
	require.NoError(t, err)
	require.NotNil(t, vault)

	files, err := vault.files()
	require.NoError(t, err)
	require.Len(t, files, 1)

	vf := files[0].(*mqlAnsibleVaultFile)
	assert.Equal(t, "1.1", vf.Format.Data)
	assert.Equal(t, "AES256", vf.Cipher.Data)
}
