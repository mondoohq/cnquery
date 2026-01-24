// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/fsutil"
	"go.mondoo.com/cnquery/v12/types"
)

// Default paths where Chrome stores extensions across different platforms and browsers
var defaultChromePaths = []string{
	// Linux - Google Chrome
	"/home/*/.config/google-chrome/*/Extensions",
	"/home/*/.config/google-chrome-beta/*/Extensions",
	"/home/*/.config/google-chrome-unstable/*/Extensions",
	// Linux - Chromium
	"/home/*/.config/chromium/*/Extensions",
	// macOS - Google Chrome
	"/Users/*/Library/Application Support/Google/Chrome/*/Extensions",
	"/Users/*/Library/Application Support/Google/Chrome Beta/*/Extensions",
	"/Users/*/Library/Application Support/Google/Chrome Canary/*/Extensions",
	// macOS - Chromium
	"/Users/*/Library/Application Support/Chromium/*/Extensions",
	// Windows - Google Chrome
	"C:\\Users\\*\\AppData\\Local\\Google\\Chrome\\User Data\\*\\Extensions",
	"C:\\Users\\*\\AppData\\Local\\Google\\Chrome Beta\\User Data\\*\\Extensions",
	"C:\\Users\\*\\AppData\\Local\\Google\\Chrome SxS\\User Data\\*\\Extensions",
	// Windows - Chromium
	"C:\\Users\\*\\AppData\\Local\\Chromium\\User Data\\*\\Extensions",
}

// chromeManifest represents the structure of a Chrome extension manifest.json
type chromeManifest struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	ManifestVersion int      `json:"manifest_version"`
	Permissions     []string `json:"permissions"`
}

func (c *mqlChrome) id() (string, error) {
	return "chrome", nil
}

func (c *mqlChrome) paths() ([]any, error) {
	result := make([]any, len(defaultChromePaths))
	for i, p := range defaultChromePaths {
		result[i] = p
	}
	return result, nil
}

func (c *mqlChrome) extensions() ([]any, error) {
	conn := c.MqlRuntime.Connection.(shared.Connection)
	fs := conn.FileSystem()
	afs := &afero.Afero{Fs: fs}

	extensions := []any{}
	seen := make(map[string]bool)

	log.Debug().Msg("searching for Chrome extensions in default locations")

	err := fsutil.WalkGlob(fs, defaultChromePaths, func(fs afero.Fs, extensionsDir string) error {
		// extensionsDir is something like /home/user/.config/google-chrome/Default/Extensions
		// Extract profile name from the path
		profile := extractChromeProfile(extensionsDir)

		// List all extension directories (each one is an extension ID)
		entries, err := afs.ReadDir(extensionsDir)
		if err != nil {
			log.Debug().Err(err).Str("path", extensionsDir).Msg("could not read extensions directory")
			return nil
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			extensionID := entry.Name()
			// Skip the "Temp" directory that Chrome sometimes creates
			if extensionID == "Temp" {
				continue
			}

			// Each extension has version subdirectories, find the latest one
			extPath := filepath.Join(extensionsDir, extensionID)
			versionDirs, err := afs.ReadDir(extPath)
			if err != nil {
				log.Debug().Err(err).Str("path", extPath).Msg("could not read extension directory")
				continue
			}

			// Find the latest version directory (or just use the first one found)
			var latestVersionDir string
			for _, vd := range versionDirs {
				if vd.IsDir() && !strings.HasPrefix(vd.Name(), ".") {
					latestVersionDir = vd.Name()
					// Continue to find the last one (usually highest version)
				}
			}

			if latestVersionDir == "" {
				continue
			}

			manifestPath := filepath.Join(extPath, latestVersionDir, "manifest.json")

			// Create a unique key to avoid duplicates
			uniqueKey := extensionID + "|" + profile
			if seen[uniqueKey] {
				continue
			}
			seen[uniqueKey] = true

			// Read and parse manifest.json
			manifest, err := readChromeManifest(afs, manifestPath)
			if err != nil {
				log.Debug().Err(err).Str("path", manifestPath).Msg("could not read manifest.json")
				continue
			}

			// Resolve localized names if needed
			name := manifest.Name
			if strings.HasPrefix(name, "__MSG_") {
				localizedName := resolveChromei18n(afs, filepath.Join(extPath, latestVersionDir), name)
				if localizedName != "" {
					name = localizedName
				}
			}

			description := manifest.Description
			if strings.HasPrefix(description, "__MSG_") {
				localizedDesc := resolveChromei18n(afs, filepath.Join(extPath, latestVersionDir), description)
				if localizedDesc != "" {
					description = localizedDesc
				}
			}

			// Convert permissions to []any
			perms := make([]any, len(manifest.Permissions))
			for i, p := range manifest.Permissions {
				perms[i] = p
			}

			ext, err := CreateResource(c.MqlRuntime, "chrome.extension", map[string]*llx.RawData{
				"__id":            llx.StringData(extensionID + "|" + profile),
				"identifier":      llx.StringData(extensionID),
				"name":            llx.StringData(name),
				"version":         llx.StringData(manifest.Version),
				"description":     llx.StringData(description),
				"manifestVersion": llx.IntData(int64(manifest.ManifestVersion)),
				"permissions":     llx.ArrayData(perms, types.String),
				"path":            llx.StringData(filepath.Join(extPath, latestVersionDir)),
				"profile":         llx.StringData(profile),
			})
			if err != nil {
				log.Debug().Err(err).Str("extension", extensionID).Msg("could not create extension resource")
				continue
			}

			extensions = append(extensions, ext)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return extensions, nil
}

// readChromeManifest reads and parses a Chrome extension manifest.json file
func readChromeManifest(afs *afero.Afero, path string) (*chromeManifest, error) {
	data, err := afs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest chromeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// extractChromeProfile extracts the Chrome profile name from an extensions path
// e.g., "/home/user/.config/google-chrome/Default/Extensions" -> "Default"
// e.g., "/home/user/.config/google-chrome/Profile 1/Extensions" -> "Profile 1"
func extractChromeProfile(extensionsDir string) string {
	// The profile name is the parent directory of "Extensions"
	dir := filepath.Dir(extensionsDir)
	return filepath.Base(dir)
}

// resolveChromei18n attempts to resolve a Chrome extension localized message
// Messages are in the format __MSG_messageName__
func resolveChromei18n(afs *afero.Afero, extDir string, msgKey string) string {
	// Extract message name from __MSG_messageName__
	if !strings.HasPrefix(msgKey, "__MSG_") || !strings.HasSuffix(msgKey, "__") {
		return ""
	}
	msgName := strings.TrimPrefix(strings.TrimSuffix(msgKey, "__"), "__MSG_")

	// Try common locales in order of preference
	locales := []string{"en", "en_US", "en_GB"}

	for _, locale := range locales {
		messagesPath := filepath.Join(extDir, "_locales", locale, "messages.json")
		data, err := afs.ReadFile(messagesPath)
		if err != nil {
			continue
		}

		var messages map[string]struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(data, &messages); err != nil {
			continue
		}

		// Message keys are case-insensitive in Chrome
		for key, val := range messages {
			if strings.EqualFold(key, msgName) {
				return val.Message
			}
		}
	}

	return ""
}
