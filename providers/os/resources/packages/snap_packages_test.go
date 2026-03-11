// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

type snapCommandResult struct {
	stdout     string
	stderr     string
	exitStatus int
	err        error
}

type snapTestConnection struct {
	fs           afero.Fs
	capabilities shared.Capabilities
	asset        *inventory.Asset
	commands     map[string]snapCommandResult
}

func (c *snapTestConnection) ID() uint32 {
	return 0
}

func (c *snapTestConnection) ParentID() uint32 {
	return 0
}

func (c *snapTestConnection) RunCommand(command string) (*shared.Command, error) {
	result, ok := c.commands[command]
	if !ok {
		return &shared.Command{
			Command:    command,
			Stdout:     bytes.NewBuffer(nil),
			Stderr:     bytes.NewBufferString("command not found"),
			ExitStatus: 1,
		}, nil
	}

	if result.err != nil {
		return nil, result.err
	}

	return &shared.Command{
		Command:    command,
		Stdout:     bytes.NewBufferString(result.stdout),
		Stderr:     bytes.NewBufferString(result.stderr),
		ExitStatus: result.exitStatus,
	}, nil
}

func (c *snapTestConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, os.ErrNotExist
}

func (c *snapTestConnection) FileSystem() afero.Fs {
	return c.fs
}

func (c *snapTestConnection) Name() string {
	return "snap-test"
}

func (c *snapTestConnection) Type() shared.ConnectionType {
	if c.capabilities.Has(shared.Capability_RunCommand) {
		return shared.Type_SSH
	}

	return shared.Type_FileSystem
}

func (c *snapTestConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *snapTestConnection) UpdateAsset(asset *inventory.Asset) {
	c.asset = asset
}

func (c *snapTestConnection) Capabilities() shared.Capabilities {
	return c.capabilities
}

type recordingFs struct {
	afero.Fs
	accesses []string
}

func (fs *recordingFs) Open(name string) (afero.File, error) {
	fs.accesses = append(fs.accesses, "open:"+name)
	return fs.Fs.Open(name)
}

func (fs *recordingFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	fs.accesses = append(fs.accesses, "openfile:"+name)
	return fs.Fs.OpenFile(name, flag, perm)
}

func (fs *recordingFs) Stat(name string) (os.FileInfo, error) {
	fs.accesses = append(fs.accesses, "stat:"+name)
	return fs.Fs.Stat(name)
}

func newSnapPkgManagerForTest(fs afero.Fs, capabilities shared.Capabilities, commands map[string]snapCommandResult) *SnapPkgManager {
	platform := &inventory.Platform{
		Name:    "ubuntu",
		Version: "22.04",
		Arch:    "amd64",
		Family:  []string{"debian", "linux", "unix", "os"},
	}

	asset := &inventory.Asset{Platform: platform}

	return &SnapPkgManager{
		conn: &snapTestConnection{
			fs:           fs,
			capabilities: capabilities,
			asset:        asset,
			commands:     commands,
		},
		platform: platform,
	}
}

func newSnapBasePathFs(t *testing.T) (afero.Fs, string) {
	t.Helper()

	root := t.TempDir()
	return afero.NewBasePathFs(afero.NewOsFs(), root), root
}

func hostPath(root string, logicalPath string) string {
	return filepath.Join(root, strings.TrimPrefix(logicalPath, "/"))
}

func writeTestFile(t *testing.T, root string, logicalPath string, content string) {
	t.Helper()

	fullPath := hostPath(root, logicalPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
}

func writeSnapManifest(t *testing.T, root string, logicalPath string, name string, version string, description string, arch string) {
	t.Helper()

	content := fmt.Sprintf("name: %s\nversion: %s\ndescription: %s\narchitectures:\n  - %s\n", name, version, description, arch)
	writeTestFile(t, root, logicalPath, content)
}

func TestParseSnapMeta(t *testing.T) {
	spm := newSnapPkgManagerForTest(afero.NewMemMapFs(), shared.Capability_None, nil)

	manifestFile, err := os.Open("testdata/snap.yaml")
	require.NoError(t, err)
	defer manifestFile.Close()

	pkg, err := spm.parseSnapManifest(manifestFile)
	require.NoError(t, err)

	assert.Equal(t, "dbgate", pkg.Name)
	assert.Equal(t, "6.1.0", pkg.Version)
	assert.Equal(t, SnapPkgFormat, pkg.Format)
	assert.Contains(t, pkg.Description, "database")
	assert.Equal(t, "pkg:snap/ubuntu/dbgate@6.1.0?arch=amd64", pkg.PUrl)
}

func TestParseSnapListOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []snapListEntry
	}{
		{
			name:     "header only returns empty list",
			output:   "Name Version Rev Tracking Publisher Notes\n",
			expected: []snapListEntry{},
		},
		{
			name:     "empty output returns empty list",
			output:   "",
			expected: []snapListEntry{},
		},
		{
			name: "parses rows and ignores multi-word notes",
			output: strings.Join([]string{
				"Name Version Rev Tracking Publisher Notes",
				"firefox 121.0 42 latest/stable canonical** -",
				"snap-store 1.2 7 latest/stable canonical** classic disabled",
			}, "\n"),
			expected: []snapListEntry{
				{name: "firefox", version: "121.0", rev: "42"},
				{name: "snap-store", version: "1.2", rev: "7"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseSnapListOutput(strings.NewReader(tt.output))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, entries)
		})
	}
}

func TestSnapPkgManagerList_NoSnapDirectoryReturnsEmpty(t *testing.T) {
	fs, _ := newSnapBasePathFs(t)
	spm := newSnapPkgManagerForTest(fs, shared.Capability_None, nil)

	pkgs, err := spm.List()
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestSnapPkgManagerList_UsesCLIManifestEnrichment(t *testing.T) {
	fs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/42/meta/snap.yaml", "firefox", "121.0", "Firefox browser", "amd64")
	writeSnapManifest(t, root, "/snap/firefox/99/meta/snap.yaml", "firefox", "999.0", "Disabled revision", "amd64")
	writeSnapManifest(t, root, "/snap/snap-store/7/meta/snap.yaml", "snap-store", "1.2", "Snap Store", "amd64")

	spm := newSnapPkgManagerForTest(fs, shared.Capability_RunCommand, map[string]snapCommandResult{
		"snap list": {
			stdout: strings.Join([]string{
				"Name Version Rev Tracking Publisher Notes",
				"firefox 121.0 42 latest/stable canonical** -",
				"snap-store 1.2 7 latest/stable canonical** classic disabled",
			}, "\n"),
		},
	})

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 2)

	firefox := findPkg(pkgs, "firefox")
	assert.Equal(t, "121.0", firefox.Version)
	assert.Equal(t, "Firefox browser", firefox.Description)
	assert.Equal(t, "amd64", firefox.Arch)
	assert.Equal(t, "pkg:snap/ubuntu/firefox@121.0?arch=amd64", firefox.PUrl)

	store := findPkg(pkgs, "snap-store")
	assert.Equal(t, "Snap Store", store.Description)
	assert.Equal(t, "1.2", store.Version)
}

func TestSnapPkgManagerList_FallsBackToFilesystemWhenCLIFails(t *testing.T) {
	fs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/42/meta/snap.yaml", "firefox", "121.0", "Firefox browser", "amd64")

	spm := newSnapPkgManagerForTest(fs, shared.Capability_RunCommand, map[string]snapCommandResult{
		"snap list": {stderr: "snap not installed", exitStatus: 1},
	})

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	assert.Equal(t, "firefox", pkgs[0].Name)
	assert.Equal(t, "121.0", pkgs[0].Version)
}

func TestSnapPkgManagerListFromFS_PrefersCurrentSymlink(t *testing.T) {
	fs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/10/meta/snap.yaml", "firefox", "10.0", "Current Firefox", "amd64")
	writeSnapManifest(t, root, "/snap/firefox/11/meta/snap.yaml", "firefox", "11.0", "Stale Firefox", "amd64")
	require.NoError(t, os.Symlink("10", hostPath(root, "/snap/firefox/current")))

	spm := newSnapPkgManagerForTest(fs, shared.Capability_None, nil)

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	assert.Equal(t, "10.0", pkgs[0].Version)
	assert.Equal(t, "Current Firefox", pkgs[0].Description)
}

func TestSnapPkgManagerListFromFS_FallsBackToHighestValidRevision(t *testing.T) {
	fs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/9/meta/snap.yaml", "firefox", "9.0", "Firefox", "amd64")
	writeTestFile(t, root, "/snap/firefox/10/meta/snap.yaml", "invalid: [\n")

	spm := newSnapPkgManagerForTest(fs, shared.Capability_None, nil)

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	assert.Equal(t, "9.0", pkgs[0].Version)
}

func TestSnapPkgManagerList_SkipsMalformedManifest(t *testing.T) {
	fs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/42/meta/snap.yaml", "firefox", "121.0", "Firefox browser", "amd64")
	writeTestFile(t, root, "/snap/broken/7/meta/snap.yaml", "not: [valid")

	spm := newSnapPkgManagerForTest(fs, shared.Capability_RunCommand, map[string]snapCommandResult{
		"snap list": {
			stdout: strings.Join([]string{
				"Name Version Rev Tracking Publisher Notes",
				"firefox 121.0 42 latest/stable canonical** -",
				"broken 1.0 7 latest/stable canonical** broken install",
			}, "\n"),
		},
	})

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	assert.Equal(t, "firefox", pkgs[0].Name)
}

func TestSnapPkgManagerListFromFS_OnlyTouchesBoundedPaths(t *testing.T) {
	baseFs, root := newSnapBasePathFs(t)
	writeSnapManifest(t, root, "/snap/firefox/123/meta/snap.yaml", "firefox", "123.0", "Firefox browser", "amd64")
	writeTestFile(t, root, "/snap/firefox/123/usr/lib/locale/en_US.UTF-8/LC_MESSAGES/ignored", "ignore me")

	recording := &recordingFs{Fs: baseFs}
	spm := newSnapPkgManagerForTest(recording, shared.Capability_None, nil)

	pkgs, err := spm.List()
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	for _, access := range recording.accesses {
		assert.NotContains(t, access, "/usr/lib/locale")
	}
}
