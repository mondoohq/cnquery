// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package aiapp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func newTestAfs() (*afero.Afero, afero.Fs) {
	fs := afero.NewMemMapFs()
	return &afero.Afero{Fs: fs}, fs
}

func writeJSON(t *testing.T, fs afero.Fs, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, path, data, 0644))
}

func detectWith(d Detector, afs *afero.Afero, home, osFamily string) []AppInfo {
	return d.Detect(DetectContext{Fs: afs, Home: home, OSFamily: osFamily})
}

// --- Desktop ---

func TestDetectDesktop_MacOS(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"

	pl := infoPlist{BundleVersion: "1.8.0", BundleID: "com.anthropic.claudefordesktop"}
	data, err := plist.Marshal(pl, plist.XMLFormat)
	require.NoError(t, err)

	require.NoError(t, fs.MkdirAll("/Applications/Claude.app/Contents", 0755))
	require.NoError(t, afero.WriteFile(fs, "/Applications/Claude.app/Contents/Info.plist", data, 0644))

	require.NoError(t, fs.MkdirAll("/Applications/Cursor.app/Contents", 0755))

	results := detectWith(&DesktopDetector{}, afs, home, "darwin")
	require.Len(t, results, 2)

	byName := map[string]AppInfo{}
	for _, r := range results {
		byName[r.Name] = r
	}

	claude := byName["Claude Desktop"]
	assert.Equal(t, "desktop", claude.Category)
	assert.Equal(t, "Anthropic", claude.Vendor)
	assert.Equal(t, "1.8.0", claude.Version)
	assert.True(t, claude.Installed)

	cursor := byName["Cursor"]
	assert.Equal(t, "desktop", cursor.Category)
	assert.Equal(t, "Anysphere", cursor.Vendor)
	assert.Equal(t, "", cursor.Version)
}

func TestDetectDesktop_NotDarwin(t *testing.T) {
	afs, _ := newTestAfs()
	results := detectWith(&DesktopDetector{}, afs, "/home/testuser", "linux")
	assert.Nil(t, results)
}

func TestDetectDesktop_NoAppsInstalled(t *testing.T) {
	afs, _ := newTestAfs()
	results := detectWith(&DesktopDetector{}, afs, "/Users/testuser", "darwin")
	assert.Empty(t, results)
}

// --- VS Code ---

func TestDetectVSCode_Extensions(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"
	extDir := filepath.Join(home, ".vscode", "extensions")

	pkg := vsCodePackageJSON{Version: "2.1.145"}
	claudeDir := filepath.Join(extDir, "anthropic.claude-code-2.1.145-darwin-arm64")
	require.NoError(t, fs.MkdirAll(claudeDir, 0755))
	writeJSON(t, fs, filepath.Join(claudeDir, "package.json"), pkg)

	copilotDir := filepath.Join(extDir, "github.copilot-1.250.0")
	require.NoError(t, fs.MkdirAll(copilotDir, 0755))
	writeJSON(t, fs, filepath.Join(copilotDir, "package.json"), vsCodePackageJSON{Version: "1.250.0"})

	results := detectWith(&VSCodeDetector{}, afs, home, "darwin")
	require.Len(t, results, 2)

	byName := map[string]AppInfo{}
	for _, r := range results {
		byName[r.Name] = r
	}

	claude := byName["Claude Code"]
	assert.Equal(t, "ide-extension", claude.Category)
	assert.Equal(t, "Anthropic", claude.Vendor)
	assert.Equal(t, "2.1.145", claude.Version)

	copilot := byName["GitHub Copilot"]
	assert.Equal(t, "ide-extension", copilot.Category)
	assert.Equal(t, "GitHub", copilot.Vendor)
	assert.Equal(t, "1.250.0", copilot.Version)
}

func TestDetectVSCode_Insiders(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"
	extDir := filepath.Join(home, ".vscode-insiders", "extensions")

	dir := filepath.Join(extDir, "continue.continue-1.0.0")
	require.NoError(t, fs.MkdirAll(dir, 0755))
	writeJSON(t, fs, filepath.Join(dir, "package.json"), vsCodePackageJSON{Version: "1.0.0"})

	results := detectWith(&VSCodeDetector{}, afs, home, "darwin")
	require.Len(t, results, 1)
	assert.Equal(t, "Continue", results[0].Name)
}

func TestDetectVSCode_NoExtensions(t *testing.T) {
	afs, _ := newTestAfs()
	results := detectWith(&VSCodeDetector{}, afs, "/Users/testuser", "darwin")
	assert.Empty(t, results)
}

func TestDetectVSCode_IgnoresNonAI(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"
	extDir := filepath.Join(home, ".vscode", "extensions")

	dir := filepath.Join(extDir, "esbenp.prettier-vscode-11.0.0")
	require.NoError(t, fs.MkdirAll(dir, 0755))

	results := detectWith(&VSCodeDetector{}, afs, home, "darwin")
	assert.Empty(t, results)
}

// --- Chrome ---

func TestDetectChrome_Extensions(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"

	profileDir := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	// Claude extension
	claudeExtID := "agoklgmhkadcfnfmgfafjhpcibpciool"
	versionDir := filepath.Join(profileDir, "Extensions", claudeExtID, "1.0.0_0")
	require.NoError(t, fs.MkdirAll(versionDir, 0755))
	writeJSON(t, fs, filepath.Join(versionDir, "manifest.json"), chromeManifest{Name: "Claude", Version: "1.0.0"})

	results := detectWith(&ChromeDetector{}, afs, home, "darwin")
	require.Len(t, results, 1)

	assert.Equal(t, "Claude", results[0].Name)
	assert.Equal(t, "browser-extension", results[0].Category)
	assert.Equal(t, "Anthropic", results[0].Vendor)
	assert.Equal(t, "1.0.0", results[0].Version)
}

func TestDetectChrome_MultipleProfiles(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"
	base := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")

	require.NoError(t, fs.MkdirAll(filepath.Join(base, "Default"), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(base, "Profile 1"), 0755))

	chatgptID := "jjfacpkknndmilbakcaefalmfckoklcp"
	vDir := filepath.Join(base, "Default", "Extensions", chatgptID, "1.0_0")
	require.NoError(t, fs.MkdirAll(vDir, 0755))
	writeJSON(t, fs, filepath.Join(vDir, "manifest.json"), chromeManifest{Name: "ChatGPT", Version: "1.0"})

	vDir2 := filepath.Join(base, "Profile 1", "Extensions", chatgptID, "1.0_0")
	require.NoError(t, fs.MkdirAll(vDir2, 0755))
	writeJSON(t, fs, filepath.Join(vDir2, "manifest.json"), chromeManifest{Name: "ChatGPT", Version: "1.0"})

	results := detectWith(&ChromeDetector{}, afs, home, "darwin")
	require.Len(t, results, 1, "should deduplicate across profiles")
}

func TestDetectChrome_NoBrowser(t *testing.T) {
	afs, _ := newTestAfs()
	results := detectWith(&ChromeDetector{}, afs, "/Users/testuser", "darwin")
	assert.Empty(t, results)
}

// --- DetectAll ---

func TestDetectAll(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/Users/testuser"

	require.NoError(t, fs.MkdirAll("/Applications/Claude.app/Contents", 0755))

	extDir := filepath.Join(home, ".vscode", "extensions")
	dir := filepath.Join(extDir, "github.copilot-1.0.0")
	require.NoError(t, fs.MkdirAll(dir, 0755))
	writeJSON(t, fs, filepath.Join(dir, "package.json"), vsCodePackageJSON{Version: "1.0.0"})

	results := DetectAll(afs, home, "darwin")
	assert.GreaterOrEqual(t, len(results), 2)

	categories := map[string]bool{}
	for _, r := range results {
		categories[r.Category] = true
	}
	assert.True(t, categories["desktop"])
	assert.True(t, categories["ide-extension"])
}
