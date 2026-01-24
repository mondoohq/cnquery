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

// browserNames maps the path component to a human-readable browser name
var browserNames = map[string]string{
	// Linux - Chrome variants
	"google-chrome":          "Google Chrome",
	"google-chrome-beta":     "Google Chrome Beta",
	"google-chrome-unstable": "Google Chrome Dev",
	"chromium":               "Chromium",
	// Linux - Edge
	"microsoft-edge":      "Microsoft Edge",
	"microsoft-edge-beta": "Microsoft Edge Beta",
	"microsoft-edge-dev":  "Microsoft Edge Dev",
	// Linux - Other Chromium-based browsers
	"BraveSoftware/Brave-Browser": "Brave",
	"vivaldi":                     "Vivaldi",
	"opera":                       "Opera",
	// macOS - Chrome variants
	"Google/Chrome":        "Google Chrome",
	"Google/Chrome Beta":   "Google Chrome Beta",
	"Google/Chrome Canary": "Google Chrome Canary",
	"Chromium":             "Chromium",
	// macOS - Edge
	"Microsoft Edge":      "Microsoft Edge",
	"Microsoft Edge Beta": "Microsoft Edge Beta",
	"Microsoft Edge Dev":  "Microsoft Edge Dev",
	// macOS - Other Chromium-based browsers
	"Vivaldi":                 "Vivaldi",
	"com.operasoftware.Opera": "Opera",
	// Windows (case-insensitive matching needed)
	"google/chrome":                "Google Chrome",
	"google/chrome beta":           "Google Chrome Beta",
	"google/chrome sxs":            "Google Chrome Canary",
	"microsoft/edge":               "Microsoft Edge",
	"microsoft/edge beta":          "Microsoft Edge Beta",
	"microsoft/edge dev":           "Microsoft Edge Dev",
	"bravesoftware/brave-browser":  "Brave",
	"opera software/opera stable":  "Opera",
}

// Search configurations for different platforms
// Covers Chrome, Chromium, Edge, Brave, Vivaldi, and Opera
// Uses files.find with name="manifest.json" then filters results by path pattern
var chromeSearchConfigs = map[string][]chromeSearchConfig{
	"linux": {
		{
			basePath:    "/home",
			pathPattern: regexp.MustCompile(`/\.config/(google-chrome|google-chrome-beta|google-chrome-unstable|chromium|microsoft-edge|microsoft-edge-beta|microsoft-edge-dev|BraveSoftware/Brave-Browser|vivaldi|opera)/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       10,
		},
	},
	"darwin": {
		{
			basePath:    "/Users",
			pathPattern: regexp.MustCompile(`/Library/Application Support/(Google/Chrome|Google/Chrome Beta|Google/Chrome Canary|Chromium|Microsoft Edge|Microsoft Edge Beta|Microsoft Edge Dev|BraveSoftware/Brave-Browser|Vivaldi|com\.operasoftware\.Opera)/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       11,
		},
	},
	"windows": {
		{
			basePath:    "C:\\Users",
			pathPattern: regexp.MustCompile(`(?i)/AppData/Local/(Google/Chrome|Google/Chrome Beta|Google/Chrome SxS|Microsoft/Edge|Microsoft/Edge Beta|Microsoft/Edge Dev|BraveSoftware/Brave-Browser|Vivaldi|Opera Software/Opera Stable)/User Data/([^/]+)/Extensions/([^/]+)/[^/]+/manifest\.json$`),
			depth:       11,
		},
	},
}

// Default paths for display (informational only)
var defaultChromePaths = []string{
	// Linux - Chrome/Chromium
	"/home/*/.config/google-chrome/*/Extensions",
	"/home/*/.config/chromium/*/Extensions",
	// Linux - Edge
	"/home/*/.config/microsoft-edge/*/Extensions",
	// Linux - Other browsers
	"/home/*/.config/BraveSoftware/Brave-Browser/*/Extensions",
	"/home/*/.config/vivaldi/*/Extensions",
	// macOS - Chrome/Chromium
	"/Users/*/Library/Application Support/Google/Chrome/*/Extensions",
	"/Users/*/Library/Application Support/Chromium/*/Extensions",
	// macOS - Edge
	"/Users/*/Library/Application Support/Microsoft Edge/*/Extensions",
	// macOS - Other browsers
	"/Users/*/Library/Application Support/BraveSoftware/Brave-Browser/*/Extensions",
	"/Users/*/Library/Application Support/Vivaldi/*/Extensions",
	// Windows - Chrome/Chromium
	"C:\\Users\\*\\AppData\\Local\\Google\\Chrome\\User Data\\*\\Extensions",
	// Windows - Edge
	"C:\\Users\\*\\AppData\\Local\\Microsoft\\Edge\\User Data\\*\\Extensions",
	// Windows - Other browsers
	"C:\\Users\\*\\AppData\\Local\\BraveSoftware\\Brave-Browser\\User Data\\*\\Extensions",
	"C:\\Users\\*\\AppData\\Local\\Vivaldi\\User Data\\*\\Extensions",
}

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

	log.Debug().Str("platform", platformKey).Msg("searching for browser extensions")

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

			// Filter to only browser extension manifest files using the path pattern
			matches := config.pathPattern.FindStringSubmatch(normalizedPath)
			if matches == nil {
				continue
			}

			// Extract browser variant, profile and extension ID from regex groups
			// Group 1: browser variant, Group 2: profile, Group 3: extension ID
			browserVariant := matches[1]
			profile := matches[2]
			extensionID := matches[3]

			// Get human-readable browser name
			browser := getBrowserName(browserVariant)

			// Create unique key including browser to avoid cross-browser deduplication
			// Same extension in Chrome and Edge should be listed separately
			uniqueKey := browser + "|" + profile + "|" + extensionID
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
				"browser":         llx.StringData(browser),
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

// getBrowserName returns a human-readable browser name from the path component
func getBrowserName(variant string) string {
	// Try exact match first
	if name, ok := browserNames[variant]; ok {
		return name
	}
	// Try case-insensitive match (for Windows paths)
	lowerVariant := strings.ToLower(variant)
	if name, ok := browserNames[lowerVariant]; ok {
		return name
	}
	// Fallback to the variant itself with some cleanup
	return strings.ReplaceAll(variant, "/", " ")
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
