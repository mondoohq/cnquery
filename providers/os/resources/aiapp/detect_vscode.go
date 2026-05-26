// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package aiapp

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// VSCodeDetector discovers installed AI-related VS Code extensions.
type VSCodeDetector struct{}

type knownExtension struct {
	ID     string // publisher.name (lowercase)
	Name   string
	Vendor string
}

var aiVSCodeExtensions = []knownExtension{
	{ID: "anthropic.claude-code", Name: "Claude Code", Vendor: "Anthropic"},
	{ID: "github.copilot", Name: "GitHub Copilot", Vendor: "GitHub"},
	{ID: "github.copilot-chat", Name: "GitHub Copilot Chat", Vendor: "GitHub"},
	{ID: "continue.continue", Name: "Continue", Vendor: "Continue"},
	{ID: "sourcegraph.cody-ai", Name: "Cody", Vendor: "Sourcegraph"},
	{ID: "tabnine.tabnine-vscode", Name: "Tabnine", Vendor: "Tabnine"},
	{ID: "codeium.codeium", Name: "Codeium", Vendor: "Codeium"},
	{ID: "supermaven.supermaven", Name: "Supermaven", Vendor: "Supermaven"},
	{ID: "amazonwebservices.aws-toolkit-vscode", Name: "AWS Toolkit (CodeWhisperer)", Vendor: "Amazon"},
	{ID: "saoudrizwan.claude-dev", Name: "Cline", Vendor: "Cline"},
	{ID: "rooveterinaryinc.roo-cline", Name: "Roo Code", Vendor: "Roo Veterinary"},
	{ID: "codegpt.codegpt", Name: "CodeGPT", Vendor: "CodeGPT"},
	{ID: "blackboxapp.blackbox", Name: "Blackbox AI", Vendor: "Blackbox"},
	{ID: "googlecloudtools.cloudcode", Name: "Google Cloud Code (Gemini)", Vendor: "Google"},
	{ID: "cursor.cursor", Name: "Cursor", Vendor: "Anysphere"},
}

type vsCodePackageJSON struct {
	Version string `json:"version"`
}

func (d *VSCodeDetector) Detect(ctx DetectContext) []AppInfo {
	extDirs := []string{
		filepath.Join(ctx.Home, ".vscode", "extensions"),
		filepath.Join(ctx.Home, ".vscode-insiders", "extensions"),
	}

	idMap := make(map[string]knownExtension, len(aiVSCodeExtensions))
	for _, ext := range aiVSCodeExtensions {
		idMap[ext.ID] = ext
	}

	var results []AppInfo
	for _, extDir := range extDirs {
		entries, err := ctx.Fs.ReadDir(extDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dirName := strings.ToLower(entry.Name())
			for id, ext := range idMap {
				if !strings.HasPrefix(dirName, id+"-") {
					continue
				}

				ai := AppInfo{
					Name:      ext.Name,
					Category:  "ide-extension",
					Vendor:    ext.Vendor,
					Path:      filepath.Join(extDir, entry.Name()),
					Installed: true,
					UpdatedAt: entry.ModTime(),
				}

				pkgPath := filepath.Join(extDir, entry.Name(), "package.json")
				if data, readErr := ctx.Fs.ReadFile(pkgPath); readErr == nil {
					var pkg vsCodePackageJSON
					if json.Unmarshal(data, &pkg) == nil && pkg.Version != "" {
						ai.Version = pkg.Version
					}
				}

				results = append(results, ai)
				break
			}
		}
	}
	return results
}
