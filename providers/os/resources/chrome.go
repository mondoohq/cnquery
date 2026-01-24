// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/types"
)

// Chrome extension search configurations per platform
type chromeSearchConfig struct {
	basePath    string
	pathPattern *regexp.Regexp
	depth       int64
}

// Search configurations for different platforms
// Uses files.find with name="manifest.json" then filters results by path pattern
var chromeSearchConfigs = map[string][]chromeSearchConfig{
	"linux": {
		{
			basePath:    "/home",
			pathPattern: regexp.MustCompile(`/\.config/(google-chrome|google-chrome-beta|google-chrome-unstable|chromium)/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       9,
		},
	},
	"darwin": {
		{
			basePath:    "/Users",
			pathPattern: regexp.MustCompile(`/Library/Application Support/(Google/Chrome|Google/Chrome Beta|Google/Chrome Canary|Chromium)/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       11,
		},
	},
	"windows": {
		{
			basePath:    "C:\\Users",
			pathPattern: regexp.MustCompile(`(?i)/AppData/Local/(Google/Chrome|Google/Chrome Beta|Google/Chrome SxS|Chromium)/User Data/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       11,
		},
	},
}

// Default paths for display (informational only)
var defaultChromePaths = []string{
	// Linux
	"/home/*/.config/google-chrome/*/Extensions",
	"/home/*/.config/chromium/*/Extensions",
	// macOS
	"/Users/*/Library/Application Support/Google/Chrome/*/Extensions",
	"/Users/*/Library/Application Support/Chromium/*/Extensions",
	// Windows
	"C:\\Users\\*\\AppData\\Local\\Google\\Chrome\\User Data\\*\\Extensions",
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
	pf := conn.Asset().Platform
	if pf == nil {
		return nil, nil
	}

	// Determine platform family
	var platformKey string
	switch {
	case pf.IsFamily("linux"):
		platformKey = "linux"
	case pf.IsFamily("darwin"):
		platformKey = "darwin"
	case pf.IsFamily("windows"):
		platformKey = "windows"
	default:
		log.Debug().Str("platform", pf.Name).Msg("unsupported platform for Chrome extension detection")
		return []any{}, nil
	}

	configs, ok := chromeSearchConfigs[platformKey]
	if !ok {
		return []any{}, nil
	}

	extensions := []any{}
	seen := make(map[string]bool)

	fs := conn.FileSystem()
	afs := &afero.Afero{Fs: fs}

	log.Debug().Str("platform", platformKey).Msg("searching for Chrome extensions")

	for _, config := range configs {
		// Use files.find resource for efficient searching
		// Uses find command on Unix, PowerShell on Windows - single RTT for discovery
		filesFind, err := CreateResource(c.MqlRuntime, "files.find", map[string]*llx.RawData{
			"from":  llx.StringData(config.basePath),
			"name":  llx.StringData("manifest.json"),
			"type":  llx.StringData("f"),
			"depth": llx.IntData(config.depth),
		})
		if err != nil {
			log.Debug().Err(err).Str("basePath", config.basePath).Msg("could not create files.find resource")
			continue
		}

		ff := filesFind.(*mqlFilesFind)
		fileList := ff.GetList()
		if fileList.Error != nil {
			log.Debug().Err(fileList.Error).Str("basePath", config.basePath).Msg("files.find failed")
			continue
		}

		for _, f := range fileList.Data {
			file := f.(*mqlFile)
			manifestPath := file.GetPath()
			if manifestPath.Error != nil {
				continue
			}

			// Normalize path for regex matching (use forward slashes)
			normalizedPath := filepath.ToSlash(manifestPath.Data)

			// Filter to only Chrome extension manifest files using the path pattern
			matches := config.pathPattern.FindStringSubmatch(normalizedPath)
			if matches == nil {
				continue
			}

			// Extract profile and extension ID from regex groups
			// Group 1: browser variant, Group 2: profile, Group 3: extension ID
			profile := matches[2]
			extensionID := matches[3]

			// Create unique key to avoid duplicates
			uniqueKey := extensionID + "|" + profile
			if seen[uniqueKey] {
				continue
			}
			seen[uniqueKey] = true

			// Read and parse manifest.json
			manifest, err := readChromeManifest(afs, manifestPath.Data)
			if err != nil {
				log.Debug().Err(err).Str("path", manifestPath.Data).Msg("could not read manifest.json")
				continue
			}

			// Resolve localized names if needed
			extDir := filepath.Dir(manifestPath.Data)
			name := manifest.Name
			if strings.HasPrefix(name, "__MSG_") {
				if localizedName := resolveChromei18n(afs, extDir, name); localizedName != "" {
					name = localizedName
				}
			}

			description := manifest.Description
			if strings.HasPrefix(description, "__MSG_") {
				if localizedDesc := resolveChromei18n(afs, extDir, description); localizedDesc != "" {
					description = localizedDesc
				}
			}

			// Convert permissions to []any
			perms := make([]any, len(manifest.Permissions))
			for i, p := range manifest.Permissions {
				perms[i] = p
			}

			ext, err := CreateResource(c.MqlRuntime, "chrome.extension", map[string]*llx.RawData{
				"__id":            llx.StringData(uniqueKey),
				"identifier":      llx.StringData(extensionID),
				"name":            llx.StringData(name),
				"version":         llx.StringData(manifest.Version),
				"description":     llx.StringData(description),
				"manifestVersion": llx.IntData(int64(manifest.ManifestVersion)),
				"permissions":     llx.ArrayData(perms, types.String),
				"path":            llx.StringData(extDir),
				"profile":         llx.StringData(profile),
			})
			if err != nil {
				log.Debug().Err(err).Str("extension", extensionID).Msg("could not create extension resource")
				continue
			}

			extensions = append(extensions, ext)
		}
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
