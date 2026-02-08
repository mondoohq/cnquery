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
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

// browserConfig defines a browser's extension path configuration
type browserConfig struct {
	name        string // Human-readable browser name
	relPath     string // Path relative to user home directory
	profilePath string // Additional path to profiles (e.g., "User Data" for Chrome on Windows)
}

// browserConfigs maps platform to list of browser configurations
// Each browser config specifies the relative path from user home to the browser data directory
var browserConfigs = map[string][]browserConfig{
	"linux": {
		{name: "Google Chrome", relPath: ".config/google-chrome"},
		{name: "Google Chrome Beta", relPath: ".config/google-chrome-beta"},
		{name: "Google Chrome Dev", relPath: ".config/google-chrome-unstable"},
		{name: "Chromium", relPath: ".config/chromium"},
		{name: "Microsoft Edge", relPath: ".config/microsoft-edge"},
		{name: "Microsoft Edge Beta", relPath: ".config/microsoft-edge-beta"},
		{name: "Microsoft Edge Dev", relPath: ".config/microsoft-edge-dev"},
		{name: "Brave", relPath: ".config/BraveSoftware/Brave-Browser"},
		{name: "Vivaldi", relPath: ".config/vivaldi"},
		{name: "Opera", relPath: ".config/opera"},
	},
	"darwin": {
		{name: "Google Chrome", relPath: "Library/Application Support/Google/Chrome"},
		{name: "Google Chrome Beta", relPath: "Library/Application Support/Google/Chrome Beta"},
		{name: "Google Chrome Canary", relPath: "Library/Application Support/Google/Chrome Canary"},
		{name: "Chromium", relPath: "Library/Application Support/Chromium"},
		{name: "Microsoft Edge", relPath: "Library/Application Support/Microsoft Edge"},
		{name: "Microsoft Edge Beta", relPath: "Library/Application Support/Microsoft Edge Beta"},
		{name: "Microsoft Edge Dev", relPath: "Library/Application Support/Microsoft Edge Dev"},
		{name: "Brave", relPath: "Library/Application Support/BraveSoftware/Brave-Browser"},
		{name: "Vivaldi", relPath: "Library/Application Support/Vivaldi"},
		{name: "Opera", relPath: "Library/Application Support/com.operasoftware.Opera"},
		{name: "Perplexity Comet", relPath: "Library/Application Support/Comet"},
		{name: "ChatGPT Atlas", relPath: "Library/Application Support/ChatGPT"},
	},
	"windows": {
		// AppData\Local browsers
		{name: "Google Chrome", relPath: "AppData/Local/Google/Chrome", profilePath: "User Data"},
		{name: "Google Chrome Beta", relPath: "AppData/Local/Google/Chrome Beta", profilePath: "User Data"},
		{name: "Google Chrome Canary", relPath: "AppData/Local/Google/Chrome SxS", profilePath: "User Data"},
		{name: "Microsoft Edge", relPath: "AppData/Local/Microsoft/Edge", profilePath: "User Data"},
		{name: "Microsoft Edge Beta", relPath: "AppData/Local/Microsoft/Edge Beta", profilePath: "User Data"},
		{name: "Microsoft Edge Dev", relPath: "AppData/Local/Microsoft/Edge Dev", profilePath: "User Data"},
		{name: "Brave", relPath: "AppData/Local/BraveSoftware/Brave-Browser", profilePath: "User Data"},
		{name: "Vivaldi", relPath: "AppData/Local/Vivaldi", profilePath: "User Data"},
		{name: "Perplexity Comet", relPath: "AppData/Local/Comet", profilePath: "User Data"},
		{name: "ChatGPT Atlas", relPath: "AppData/Local/OpenAI/Atlas", profilePath: "User Data"},
		// AppData\Roaming browsers (Opera has different structure)
		{name: "Opera", relPath: "AppData/Roaming/Opera Software/Opera Stable"},
	},
}

// extensionPathPattern matches Chrome extension manifest paths and extracts profile and extension ID
// Group 1: profile name, Group 2: extension ID
var extensionPathPattern = regexp.MustCompile(`/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`)

// chromeManifest represents the structure of a Chrome extension manifest.json
type chromeManifest struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	ManifestVersion int      `json:"manifest_version"`
	Permissions     []string `json:"permissions"`
	DefaultLocale   string   `json:"default_locale"`
}

func (c *mqlChrome) id() (string, error) {
	return "chrome", nil
}

func (c *mqlChrome) extensions() ([]any, error) {
	conn := c.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform
	if pf == nil {
		return nil, nil
	}

	platformKey := getPlatformKey(pf)
	if platformKey == "" {
		log.Debug().Str("platform", pf.Name).Msg("unsupported platform for Chrome extension detection")
		return []any{}, nil
	}

	configs, ok := browserConfigs[platformKey]
	if !ok {
		return []any{}, nil
	}

	// Get list of users to find their home directories
	usersResource, err := CreateResource(c.MqlRuntime, "users", map[string]*llx.RawData{})
	if err != nil {
		log.Debug().Err(err).Msg("could not get users list")
		return []any{}, nil
	}
	users := usersResource.(*mqlUsers)
	userList := users.GetList()
	if userList.Error != nil {
		log.Debug().Err(userList.Error).Msg("could not retrieve users list")
		return []any{}, nil
	}

	extensions := []any{}
	seen := make(map[string]bool)

	fs := conn.FileSystem()
	afs := &afero.Afero{Fs: fs}

	log.Debug().Str("platform", platformKey).Int("userCount", len(userList.Data)).Msg("searching for browser extensions")

	// Iterate through each user's home directory
	for _, u := range userList.Data {
		user := u.(*mqlUser)
		home := user.GetHome()
		if home.Error != nil || home.Data == "" {
			continue
		}

		// Skip system users (no valid home or special paths)
		homeDir := home.Data
		if !isValidUserHome(homeDir, platformKey) {
			continue
		}

		// Check each browser for this user
		for _, browserCfg := range configs {
			browserDir := filepath.Join(homeDir, browserCfg.relPath)
			if browserCfg.profilePath != "" {
				browserDir = filepath.Join(browserDir, browserCfg.profilePath)
			}

			// Check if browser directory exists before searching
			exists, err := afs.DirExists(browserDir)
			if err != nil || !exists {
				continue
			}

			log.Debug().Str("browser", browserCfg.name).Str("path", browserDir).Msg("found browser directory")

			// Search for extensions in this browser directory
			// Depth 5: profile/Extensions/extID/version/manifest.json
			filesFind, err := CreateResource(c.MqlRuntime, "files.find", map[string]*llx.RawData{
				"from":  llx.StringData(browserDir),
				"name":  llx.StringData("manifest.json"),
				"type":  llx.StringData("f"),
				"depth": llx.IntData(5),
			})
			if err != nil {
				log.Debug().Err(err).Str("browserDir", browserDir).Msg("could not create files.find resource")
				continue
			}

			ff := filesFind.(*mqlFilesFind)
			fileList := ff.GetList()
			if fileList.Error != nil {
				log.Debug().Err(fileList.Error).Str("browserDir", browserDir).Msg("files.find failed")
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

				// Filter to only extension manifest files using the path pattern
				matches := extensionPathPattern.FindStringSubmatch(normalizedPath)
				if matches == nil {
					continue
				}

				// Extract profile and extension ID from regex groups
				// Group 1: profile name, Group 2: extension ID
				profile := matches[1]
				extensionID := matches[2]

				// Create unique key including browser to avoid cross-browser deduplication
				// Same extension in Chrome and Edge should be listed separately
				uniqueKey := browserCfg.name + "|" + profile + "|" + extensionID
				if seen[uniqueKey] {
					// Skip duplicate (same extension, same browser, same profile)
					// This can happen if multiple versions exist - we take the first one found
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
					if localizedName := resolveChromei18n(afs, extDir, manifest.DefaultLocale, name); localizedName != "" {
						name = localizedName
					}
				}

				description := manifest.Description
				if strings.HasPrefix(description, "__MSG_") {
					if localizedDesc := resolveChromei18n(afs, extDir, manifest.DefaultLocale, description); localizedDesc != "" {
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
					"browser":         llx.StringData(browserCfg.name),
				})
				if err != nil {
					log.Debug().Err(err).Str("extension", extensionID).Msg("could not create extension resource")
					continue
				}

				extensions = append(extensions, ext)
			}
		}
	}

	return extensions, nil
}

// getPlatformKey returns the platform key for browser configs lookup
func getPlatformKey(pf interface{ IsFamily(string) bool }) string {
	switch {
	case pf.IsFamily("linux"):
		return "linux"
	case pf.IsFamily("darwin"):
		return "darwin"
	case pf.IsFamily("windows"):
		return "windows"
	default:
		return ""
	}
}

// isValidUserHome checks if a home directory is a valid user home (not a system account)
func isValidUserHome(homeDir string, platform string) bool {
	if homeDir == "" {
		return false
	}

	switch platform {
	case "linux":
		// Valid homes: /home/*, /root
		return strings.HasPrefix(homeDir, "/home/") || homeDir == "/root"
	case "darwin":
		// Valid homes: /Users/* (excluding /Users/Shared)
		return strings.HasPrefix(homeDir, "/Users/") && homeDir != "/Users/Shared"
	case "windows":
		// Valid homes: C:\Users\* (excluding system accounts)
		lowerHome := strings.ToLower(homeDir)
		if !strings.HasPrefix(lowerHome, "c:\\users\\") && !strings.HasPrefix(lowerHome, "c:/users/") {
			return false
		}
		// Exclude known system accounts
		excludedUsers := []string{"default", "public", "all users", "default user"}
		for _, excluded := range excludedUsers {
			if strings.Contains(lowerHome, "\\"+excluded) || strings.Contains(lowerHome, "/"+excluded) {
				return false
			}
		}
		return true
	default:
		return false
	}
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
func resolveChromei18n(afs *afero.Afero, extDir string, defaultLocale string, msgKey string) string {
	// Extract message name from __MSG_messageName__
	if !strings.HasPrefix(msgKey, "__MSG_") || !strings.HasSuffix(msgKey, "__") {
		return ""
	}
	msgName := strings.TrimPrefix(strings.TrimSuffix(msgKey, "__"), "__MSG_")

	// Build locale search order: default_locale first, then common fallbacks
	locales := []string{}
	if defaultLocale != "" {
		locales = append(locales, defaultLocale)
	}
	// Add common English locales as fallback
	for _, loc := range []string{"en", "en_US", "en_GB"} {
		if loc != defaultLocale {
			locales = append(locales, loc)
		}
	}

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
