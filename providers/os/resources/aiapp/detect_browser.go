// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package aiapp

import (
	"encoding/json"
	"path/filepath"
)

// ChromeDetector discovers installed AI-related Chrome/Chromium extensions.
type ChromeDetector struct{}

type knownChromeExtension struct {
	ExtID  string
	Name   string
	Vendor string
}

// Well-known Chrome extension IDs for AI tools.
var aiChromeExtensions = []knownChromeExtension{
	{ExtID: "agoklgmhkadcfnfmgfafjhpcibpciool", Name: "Claude", Vendor: "Anthropic"},
	{ExtID: "gpaiobkfhnonpkbkdallhkhfalfmcpjm", Name: "Claude for Enterprise", Vendor: "Anthropic"},
	{ExtID: "jjfacpkknndmilbakcaefalmfckoklcp", Name: "ChatGPT", Vendor: "OpenAI"},
	{ExtID: "obnaaalnokmchdoagnhmjjpchkldadbde", Name: "Perplexity", Vendor: "Perplexity AI"},
	{ExtID: "lnkdbjbjpnpjeciipoaflmpcddinpjjp", Name: "GitHub Copilot", Vendor: "GitHub"},
	{ExtID: "dgjhfomjieaadpoljlnidmbgkdffpack", Name: "Gemini", Vendor: "Google"},
	{ExtID: "pnjaknljhloefhkifoodeegdpnpljbnd", Name: "Codeium", Vendor: "Codeium"},
	{ExtID: "iadjlbflapfkpbegnpkpnhhiidalihelm", Name: "Monica AI", Vendor: "Monica"},
	{ExtID: "nfclhjlpoddfnbpkkabaoifjnnimjaog", Name: "Tabnine", Vendor: "Tabnine"},
}

type chromeManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (d *ChromeDetector) Detect(ctx DetectContext) []AppInfo {
	profileDirs := chromeProfileDirs(ctx)

	idMap := make(map[string]knownChromeExtension, len(aiChromeExtensions))
	for _, ext := range aiChromeExtensions {
		idMap[ext.ExtID] = ext
	}

	seen := make(map[string]bool)
	var results []AppInfo

	for _, profileDir := range profileDirs {
		extDir := filepath.Join(profileDir, "Extensions")
		entries, err := ctx.Fs.ReadDir(extDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ext, ok := idMap[entry.Name()]
			if !ok {
				continue
			}
			if seen[ext.ExtID] {
				continue
			}
			seen[ext.ExtID] = true

			ai := AppInfo{
				Name:      ext.Name,
				Category:  "browser-extension",
				Vendor:    ext.Vendor,
				Path:      filepath.Join(extDir, entry.Name()),
				Installed: true,
				UpdatedAt: entry.ModTime(),
			}

			versions, _ := ctx.Fs.ReadDir(filepath.Join(extDir, entry.Name()))
			if len(versions) > 0 {
				latest := versions[len(versions)-1]
				mfPath := filepath.Join(extDir, entry.Name(), latest.Name(), "manifest.json")
				if data, readErr := ctx.Fs.ReadFile(mfPath); readErr == nil {
					var mf chromeManifest
					if json.Unmarshal(data, &mf) == nil && mf.Version != "" {
						ai.Version = mf.Version
					}
				}
			}

			results = append(results, ai)
		}
	}
	return results
}

func chromeProfileDirs(ctx DetectContext) []string {
	var dirs []string
	switch ctx.OSFamily {
	case "darwin", "unix":
		base := filepath.Join(ctx.Home, "Library", "Application Support", "Google", "Chrome")
		dirs = appendChromeProfiles(ctx, base, dirs)
		// Chromium
		base = filepath.Join(ctx.Home, "Library", "Application Support", "Chromium")
		dirs = appendChromeProfiles(ctx, base, dirs)
		// Brave
		base = filepath.Join(ctx.Home, "Library", "Application Support", "BraveSoftware", "Brave-Browser")
		dirs = appendChromeProfiles(ctx, base, dirs)
		// Edge
		base = filepath.Join(ctx.Home, "Library", "Application Support", "Microsoft Edge")
		dirs = appendChromeProfiles(ctx, base, dirs)
	case "linux":
		base := filepath.Join(ctx.Home, ".config", "google-chrome")
		dirs = appendChromeProfiles(ctx, base, dirs)
		base = filepath.Join(ctx.Home, ".config", "chromium")
		dirs = appendChromeProfiles(ctx, base, dirs)
	case "windows":
		base := filepath.Join(ctx.Home, "AppData", "Local", "Google", "Chrome", "User Data")
		dirs = appendChromeProfiles(ctx, base, dirs)
		base = filepath.Join(ctx.Home, "AppData", "Local", "Microsoft", "Edge", "User Data")
		dirs = appendChromeProfiles(ctx, base, dirs)
	}
	return dirs
}

func appendChromeProfiles(ctx DetectContext, base string, dirs []string) []string {
	if _, err := ctx.Fs.Stat(filepath.Join(base, "Default")); err == nil {
		dirs = append(dirs, filepath.Join(base, "Default"))
	}
	entries, err := ctx.Fs.ReadDir(base)
	if err != nil {
		return dirs
	}
	for _, e := range entries {
		if e.IsDir() && len(e.Name()) > 8 && e.Name()[:8] == "Profile " {
			dirs = append(dirs, filepath.Join(base, e.Name()))
		}
	}
	return dirs
}
