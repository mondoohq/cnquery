// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

// vsCodeEditor represents a VS Code-based editor with its extension directory
type vsCodeEditor struct {
	dir  string // Extension directory relative to user home
	name string // Human-readable editor name
}

// VS Code extension directories relative to user home (same across platforms)
var vsCodeEditors = []vsCodeEditor{
	{".vscode/extensions", "Visual Studio Code"},
	{".vscode-insiders/extensions", "Visual Studio Code Insiders"},
	{".vscode-oss/extensions", "VSCodium"},
	{".cursor/extensions", "Cursor"},
	{".antigravity/extensions", "Antigravity"},
	{".windsurf/extensions", "Windsurf"},
	{".positron/extensions", "Positron"},
	{".kiro/extensions", "Kiro"},
}

// invalidHomeDirs contains home directories that should be skipped (system accounts)
var invalidHomeDirs = map[string]bool{
	"":                      true,
	"/var/empty":            true,
	"/nonexistent":          true,
	"/dev/null":             true,
	"/":                     true,
	"/var":                  true,
	"/usr":                  true,
	"/bin":                  true,
	"/sbin":                 true,
	"C:\\Windows\\System32": true,
	"C:\\Windows\\system32\\config\\systemprofile": true,
}

// vscodePackageJSON represents the package.json structure for VS Code extensions
type vscodePackageJSON struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Publisher   string   `json:"publisher"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	Engines     struct {
		VSCode string `json:"vscode"`
	} `json:"engines"`
}

func (c *mqlVscode) id() (string, error) {
	return "vscode", nil
}

func (c *mqlVscode) paths() ([]any, error) {
	conn := c.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	// Get all users to find home directories
	usersResource, err := CreateResource(c.MqlRuntime, "users", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	userList := usersResource.(*mqlUsers).GetList()
	if userList.Error != nil {
		return nil, userList.Error
	}

	var paths []string

	for _, u := range userList.Data {
		user := u.(*mqlUser)
		homeDir := user.GetHome().Data

		if invalidHomeDirs[homeDir] {
			continue
		}

		for _, editor := range vsCodeEditors {
			extensionsDir := filepath.Join(homeDir, editor.dir)
			// Use DirExists which is more efficient than Exists for directories
			exists, err := afs.DirExists(extensionsDir)
			if err != nil || !exists {
				continue
			}
			paths = append(paths, extensionsDir)
		}
	}

	sort.Strings(paths)

	result := make([]any, len(paths))
	for i, p := range paths {
		result[i] = p
	}
	return result, nil
}

func (c *mqlVscode) extensions() ([]any, error) {
	conn := c.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	// Get all users to find home directories
	usersResource, err := CreateResource(c.MqlRuntime, "users", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}

	userList := usersResource.(*mqlUsers).GetList()
	if userList.Error != nil {
		return nil, userList.Error
	}

	var extensions []any
	seen := make(map[string]bool)

	for _, u := range userList.Data {
		user := u.(*mqlUser)
		homeDir := user.GetHome().Data

		if invalidHomeDirs[homeDir] {
			continue
		}

		for _, editor := range vsCodeEditors {
			extensionsDir := filepath.Join(homeDir, editor.dir)

			// ReadDir will fail if directory doesn't exist - no need for separate Exists check
			entries, err := afs.ReadDir(extensionsDir)
			if err != nil {
				if !os.IsNotExist(err) {
					log.Debug().Err(err).Str("path", extensionsDir).Msg("failed to read VS Code extensions directory")
				}
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// Skip .obsolete directory
				if entry.Name() == ".obsolete" {
					continue
				}

				extPath := filepath.Join(extensionsDir, entry.Name())
				packageJSONPath := filepath.Join(extPath, "package.json")

				// ReadFile will fail if file doesn't exist - no need for separate Exists check
				pkgJSON, err := readVSCodePackageJSON(afs, packageJSONPath)
				if err != nil {
					if !os.IsNotExist(err) {
						log.Debug().Err(err).Str("path", packageJSONPath).Msg("failed to parse VS Code extension package.json")
					}
					continue
				}

				// Create unique identifier
				identifier := pkgJSON.Publisher + "." + pkgJSON.Name
				if identifier == "." {
					// Fallback to directory name if publisher/name not available
					identifier = entry.Name()
				}

				// Create unique ID for caching (including path to handle multiple installs)
				uniqueID := identifier + "|" + extPath

				if seen[uniqueID] {
					continue
				}
				seen[uniqueID] = true

				// Convert categories to []any
				categories := make([]any, len(pkgJSON.Categories))
				for i, cat := range pkgJSON.Categories {
					categories[i] = cat
				}

				// Create extension resource
				ext, err := CreateResource(c.MqlRuntime, "vscode.extension", map[string]*llx.RawData{
					"__id":          llx.StringData(uniqueID),
					"identifier":    llx.StringData(identifier),
					"name":          llx.StringData(pkgJSON.Name),
					"displayName":   llx.StringData(pkgJSON.DisplayName),
					"version":       llx.StringData(pkgJSON.Version),
					"description":   llx.StringData(pkgJSON.Description),
					"publisher":     llx.StringData(pkgJSON.Publisher),
					"editor":        llx.StringData(editor.name),
					"path":          llx.StringData(extPath),
					"vscodeVersion": llx.StringData(pkgJSON.Engines.VSCode),
					"categories":    llx.ArrayData(categories, types.String),
				})
				if err != nil {
					log.Debug().Err(err).Str("extension", identifier).Msg("failed to create VS Code extension resource")
					continue
				}
				extensions = append(extensions, ext)
			}
		}
	}

	return extensions, nil
}

func (e *mqlVscodeExtension) id() (string, error) {
	return e.Identifier.Data + "|" + e.Path.Data, nil
}

// readVSCodePackageJSON reads and parses a VS Code extension package.json file
func readVSCodePackageJSON(afs *afero.Afero, path string) (*vscodePackageJSON, error) {
	data, err := afs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg vscodePackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}
