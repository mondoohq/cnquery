// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inventory-adjacent group_vars/host_vars (e.g. inventory/group_vars/) must be
// honored in addition to the project root.
func TestInventoryAdjacentVars(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	}

	write("inventory/hosts", "[web]\nhost1\n")
	write("inventory/group_vars/web.yml", "adjacent_var: from_inventory_dir\n")
	write("inventory/host_vars/host1.yml", "host_adjacent: true\n")
	write("group_vars/all.yml", "root_var: from_root\n")

	inv, err := loadInventory(root)
	require.NoError(t, err)
	require.NotNil(t, inv)

	groups := map[string]*InventoryGroup{}
	for _, g := range inv.Groups {
		groups[g.Name] = g
	}
	assert.Equal(t, "from_inventory_dir", groups["web"].Vars["adjacent_var"])
	assert.Equal(t, "from_root", groups["all"].Vars["root_var"])

	hosts := map[string]*InventoryHost{}
	for _, h := range inv.Hosts {
		hosts[h.Name] = h
	}
	assert.Equal(t, true, hosts["host1"].Vars["host_adjacent"])
	assert.Contains(t, hosts["host1"].Groups, "web")
}

// A symlinked var file whose target lies outside the project must not be read,
// so a crafted repo cannot exfiltrate host files into the inventory model.
func TestSymlinkVarsEscapeSkipped(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.yml"), []byte("leaked: true\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "group_vars"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "inventory"), []byte("[web]\nhost1\n"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.yml"), filepath.Join(root, "group_vars", "web.yml")))

	inv, err := loadInventory(root)
	require.NoError(t, err)
	require.NotNil(t, inv)
	for _, g := range inv.Groups {
		if g.Name == "web" {
			_, leaked := g.Vars["leaked"]
			assert.False(t, leaked, "symlinked var file escaping the root must be skipped")
		}
	}
}
