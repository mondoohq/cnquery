// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package aiapp

import (
	"path/filepath"

	"howett.net/plist"
)

// DesktopDetector discovers installed AI desktop applications.
type DesktopDetector struct{}

type knownApp struct {
	Name   string
	Dir    string // directory name under /Applications (macOS) or install path
	Vendor string
}

var macApps = []knownApp{
	{Name: "Claude Desktop", Dir: "Claude.app", Vendor: "Anthropic"},
	{Name: "ChatGPT", Dir: "ChatGPT.app", Vendor: "OpenAI"},
	{Name: "Cursor", Dir: "Cursor.app", Vendor: "Anysphere"},
	{Name: "Windsurf", Dir: "Windsurf.app", Vendor: "Codeium"},
	{Name: "Ollama", Dir: "Ollama.app", Vendor: "Ollama"},
	{Name: "Jan", Dir: "Jan.app", Vendor: "Jan AI"},
	{Name: "LM Studio", Dir: "LM Studio.app", Vendor: "LM Studio"},
	{Name: "GPT4All", Dir: "GPT4All.app", Vendor: "Nomic"},
	{Name: "oMLX", Dir: "oMLX.app", Vendor: "oMLX"},
	{Name: "Msty", Dir: "Msty.app", Vendor: "Msty"},
	{Name: "Perplexity", Dir: "Perplexity.app", Vendor: "Perplexity AI"},
}

type infoPlist struct {
	BundleVersion string `plist:"CFBundleShortVersionString"`
	BundleID      string `plist:"CFBundleIdentifier"`
}

func (d *DesktopDetector) Detect(ctx DetectContext) []AppInfo {
	if ctx.OSFamily != "darwin" && ctx.OSFamily != "unix" {
		return nil
	}

	var results []AppInfo
	for _, app := range macApps {
		appDir := filepath.Join("/Applications", app.Dir)
		info, err := ctx.Fs.Stat(appDir)
		if err != nil || !info.IsDir() {
			continue
		}

		ai := AppInfo{
			Name:      app.Name,
			Category:  "desktop",
			Vendor:    app.Vendor,
			Path:      appDir,
			Installed: true,
			UpdatedAt: info.ModTime(),
		}

		plistPath := filepath.Join(appDir, "Contents", "Info.plist")
		if data, readErr := ctx.Fs.ReadFile(plistPath); readErr == nil {
			var pl infoPlist
			if _, parseErr := plist.Unmarshal(data, &pl); parseErr == nil {
				ai.Version = pl.BundleVersion
			}
		}

		results = append(results, ai)
	}
	return results
}
