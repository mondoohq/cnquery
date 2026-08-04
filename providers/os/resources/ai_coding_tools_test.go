// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRealUserHome(t *testing.T) {
	tests := []struct {
		home string
		want bool
	}{
		{"/home/alice", true},
		{"/Users/bob", true},
		{"/root", true},
		{"/usr/home/carol", true}, // FreeBSD
		{`C:\Users\dave`, true},
		{"", false},
		{"/var/lib/nobody", false},  // system
		{"/Users/Shared", false},    // shared profile
		{`C:\Users\Public`, false},  // shared profile
		{`C:\Users\Default`, false}, // system profile
		{"/nonexistent/prefix", false},
	}
	for _, tt := range tests {
		assert.Equalf(t, tt.want, isRealUserHome(tt.home), "isRealUserHome(%q)", tt.home)
	}
}

// denyFs wraps an afero.Fs and returns a permission error for any path under
// deny, simulating an unprivileged scan of another user's home.
type denyFs struct {
	afero.Fs
	deny string
}

func (d *denyFs) permErr(op, name string) error {
	return &os.PathError{Op: op, Path: name, Err: os.ErrPermission}
}

func (d *denyFs) Open(name string) (afero.File, error) {
	if strings.HasPrefix(name, d.deny) {
		return nil, d.permErr("open", name)
	}
	return d.Fs.Open(name)
}

func (d *denyFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if strings.HasPrefix(name, d.deny) {
		return nil, d.permErr("open", name)
	}
	return d.Fs.OpenFile(name, flag, perm)
}

func (d *denyFs) Stat(name string) (os.FileInfo, error) {
	if strings.HasPrefix(name, d.deny) {
		return nil, d.permErr("stat", name)
	}
	return d.Fs.Stat(name)
}

func writeSkill(t *testing.T, fs afero.Fs, dir, name string) {
	t.Helper()
	require.NoError(t, fs.MkdirAll(dir+"/"+name, 0o755))
	content := "---\nname: " + name + "\ndescription: " + name + " skill.\n---\n# " + name + "\n"
	require.NoError(t, afero.WriteFile(fs, dir+"/"+name+"/SKILL.md", []byte(content), 0o644))
}

func TestCollectSkillFilesSkipsUnreadableDir(t *testing.T) {
	mem := afero.NewMemMapFs()
	writeSkill(t, mem, "/home/alice/.cursor/skills", "readable")
	writeSkill(t, mem, "/home/bob/.cursor/skills", "hidden")

	// bob's home is unreadable (permission denied), alice's is fine.
	afs := &afero.Afero{Fs: &denyFs{Fs: mem, deny: "/home/bob"}}

	skills := collectSkillFiles(afs, []string{
		"/home/alice/.cursor/skills",
		"/home/bob/.cursor/skills",
		"/home/carol/.cursor/skills", // missing entirely
	})

	require.Len(t, skills, 1, "should return only the readable user's skill")
	assert.Equal(t, "readable", skills[0].name)
	assert.Contains(t, skills[0].source, "/home/alice/")
}

func TestCollectSkillFilesDedupsSourcePaths(t *testing.T) {
	mem := afero.NewMemMapFs()
	writeSkill(t, mem, "/home/alice/.cursor/skills", "foo")
	afs := &afero.Afero{Fs: mem}

	// same dir passed twice (e.g. default path == override path) yields one skill
	skills := collectSkillFiles(afs, []string{
		"/home/alice/.cursor/skills",
		"/home/alice/.cursor/skills",
	})
	require.Len(t, skills, 1)
}

func TestFileURIToPath(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"file:///pub/go/src/go.mondoo.com/mql", "/pub/go/src/go.mondoo.com/mql"},
		{"file:///path/with%20space/repo", "/path/with space/repo"},
		{"file:///C:/Users/dev/repo", "C:/Users/dev/repo"},
		{"vscode-remote://ssh/pub/repo", ""},
		{"", ""},
		{"/not/a/uri", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, fileURIToPath(c.uri), "uri %q", c.uri)
	}
}

func TestVscodeUserDataDir(t *testing.T) {
	assert.Equal(t, "/home/x/.config/Cursor/User", vscodeUserDataDir("/home/x", "linux", "Cursor"))
	assert.Equal(t, "/Users/x/Library/Application Support/Windsurf/User", vscodeUserDataDir("/Users/x", "darwin", "Windsurf"))
	assert.Equal(t, "/home/x/.config/Cursor/User", vscodeUserDataDir("/home/x", "", "Cursor"))
}

func TestExtractZedPaths(t *testing.T) {
	// Simulate a bincode-ish blob: length prefixes (non-printable) between paths.
	blob := []byte{0x1a, 0x00, 0x00, 0x00}
	blob = append(blob, []byte("/pub/go/src/go.mondoo.com/mql")...)
	blob = append(blob, 0x00, 0x08)
	blob = append(blob, []byte("/home/dev/other-repo")...)
	blob = append(blob, 0x00)
	// A quoted JSON-style path and some noise that must be ignored.
	blob = append(blob, []byte(`"/quoted/path/repo"`)...)
	blob = append(blob, 0x00)
	blob = append(blob, []byte("SELECT * FROM workspaces")...)

	got := extractZedPaths(blob)
	assert.Contains(t, got, "/pub/go/src/go.mondoo.com/mql")
	assert.Contains(t, got, "/home/dev/other-repo")
	assert.Contains(t, got, "/quoted/path/repo")
	// the SQL run has no leading absolute path -> not extracted
	for _, p := range got {
		assert.NotContains(t, p, "SELECT")
	}
}

func TestZedCandidatePath(t *testing.T) {
	assert.Equal(t, "/a/b", zedCandidatePath("/a/b"))
	assert.Equal(t, "/a/b", zedCandidatePath(`"/a/b"`))
	assert.Equal(t, "C:/dev/repo", zedCandidatePath("C:/dev/repo"))
	assert.Equal(t, "", zedCandidatePath("no-path-here"))
	assert.Equal(t, "", zedCandidatePath("/"))
}
