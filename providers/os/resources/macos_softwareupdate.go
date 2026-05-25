// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/resources/plist"
)

// /Library/Managed Preferences/ takes precedence over /Library/Preferences/
// — the former is populated by an MDM Configuration Profile and reflects
// effective policy, the latter is the system-default location written by
// `softwareupdate --schedule on` and friends.
var softwareUpdatePlistPaths = []string{
	"/Library/Managed Preferences/com.apple.SoftwareUpdate.plist",
	"/Library/Preferences/com.apple.SoftwareUpdate.plist",
}

type mqlMacosSoftwareupdateInternal struct {
	settingsLock    sync.Mutex
	settingsFetched bool
	settings        softwareupdateSettings
}

type softwareupdateSettings struct {
	autoCheckEnabled         bool
	autoDownloadEnabled      bool
	autoInstallMacOSUpdates  bool
	installSystemDataFiles   bool
	installSecurityResponses bool
	lastSuccessfulCheck      string
}

func (s *mqlMacosSoftwareupdate) id() (string, error) {
	return "macos.softwareupdate", nil
}

// fetchSettings reads the SoftwareUpdate preferences plist and caches
// the parsed settings on the resource. Missing files (Mac on which no
// SoftwareUpdate preference has ever been set, or a non-Darwin host)
// leave the settings struct at its zero value — every accessor will
// then read `false`/`""`, which is what an auditor wants for an
// un-policied device.
func (s *mqlMacosSoftwareupdate) fetchSettings() (softwareupdateSettings, error) {
	if s.settingsFetched {
		return s.settings, nil
	}
	s.settingsLock.Lock()
	defer s.settingsLock.Unlock()
	if s.settingsFetched {
		return s.settings, nil
	}

	parsed, err := s.readSoftwareUpdatePlist()
	if err != nil {
		return softwareupdateSettings{}, err
	}
	s.settings = parsed
	s.settingsFetched = true
	return s.settings, nil
}

func (s *mqlMacosSoftwareupdate) readSoftwareUpdatePlist() (softwareupdateSettings, error) {
	for _, path := range softwareUpdatePlistPaths {
		res, err := NewResource(s.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return softwareupdateSettings{}, err
		}
		f := res.(*mqlFile)
		content := f.GetContent()
		if content.Error != nil || content.Data == "" {
			continue
		}
		data, err := plist.Decode(bytes.NewReader([]byte(content.Data)))
		if err != nil {
			// If the managed prefs file is malformed for some reason,
			// don't give up — fall through to the system prefs file.
			continue
		}
		return parseSoftwareUpdateSettings(data), nil
	}
	// Neither file exists — return zero-valued settings.
	return softwareupdateSettings{}, nil
}

// parseSoftwareUpdateSettings extracts the well-known keys from a
// decoded SoftwareUpdate plist. Through the plist.Decode helper,
// boolean keys come back as Go bools and dates round-trip through
// JSON as RFC3339 strings, so the conversions here are straightforward.
func parseSoftwareUpdateSettings(d map[string]any) softwareupdateSettings {
	return softwareupdateSettings{
		autoCheckEnabled:         boolFromPlist(d, "AutomaticCheckEnabled"),
		autoDownloadEnabled:      boolFromPlist(d, "AutomaticDownload"),
		autoInstallMacOSUpdates:  boolFromPlist(d, "AutomaticallyInstallMacOSUpdates"),
		installSystemDataFiles:   boolFromPlist(d, "ConfigDataInstall"),
		installSecurityResponses: boolFromPlist(d, "CriticalUpdateInstall"),
		lastSuccessfulCheck:      stringFromPlist(d, "LastSuccessfulDate"),
	}
}

func (s *mqlMacosSoftwareupdate) autoCheckEnabled() (bool, error) {
	v, err := s.fetchSettings()
	return v.autoCheckEnabled, err
}

func (s *mqlMacosSoftwareupdate) autoDownloadEnabled() (bool, error) {
	v, err := s.fetchSettings()
	return v.autoDownloadEnabled, err
}

func (s *mqlMacosSoftwareupdate) autoInstallMacOSUpdates() (bool, error) {
	v, err := s.fetchSettings()
	return v.autoInstallMacOSUpdates, err
}

func (s *mqlMacosSoftwareupdate) installSystemDataFiles() (bool, error) {
	v, err := s.fetchSettings()
	return v.installSystemDataFiles, err
}

func (s *mqlMacosSoftwareupdate) installSecurityResponses() (bool, error) {
	v, err := s.fetchSettings()
	return v.installSecurityResponses, err
}

func (s *mqlMacosSoftwareupdate) lastSuccessfulCheck() (string, error) {
	v, err := s.fetchSettings()
	return v.lastSuccessfulCheck, err
}

// =============================================================================
// updates() — `softwareupdate -l --no-scan`
// =============================================================================

func (s *mqlMacosSoftwareupdate) updates() ([]any, error) {
	// --no-scan reads from the local cache rather than initiating a
	// network check. A fresh network scan can take 30+ seconds; auditing
	// shouldn't pay that cost on every query.
	res, err := NewResource(s.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("softwareupdate -l --no-scan"),
	})
	if err != nil {
		return nil, err
	}
	cmd := res.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		// On non-Darwin hosts or when softwareupdate is unavailable,
		// surface an empty list rather than failing the whole query.
		return []any{}, nil
	}

	parsed := parseSoftwareUpdateList(cmd.GetStdout().Data)
	out := make([]any, 0, len(parsed))
	for _, u := range parsed {
		entry, err := CreateResource(s.MqlRuntime, "macos.softwareupdate.entry", map[string]*llx.RawData{
			"label":       llx.StringData(u.label),
			"title":       llx.StringData(u.title),
			"version":     llx.StringData(u.version),
			"size":        llx.StringData(u.size),
			"recommended": llx.BoolData(u.recommended),
			"action":      llx.StringData(u.action),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}

func (e *mqlMacosSoftwareupdateEntry) id() (string, error) {
	if e.Label.Data == "" {
		return "", errors.New("software update entry missing label")
	}
	return "macos.softwareupdate.entry:" + e.Label.Data, nil
}

type parsedSoftwareUpdate struct {
	label       string
	title       string
	version     string
	size        string
	recommended bool
	action      string
}

// parseSoftwareUpdateList parses the text output of `softwareupdate -l`.
// The format has been stable across recent macOS versions:
//
//	Software Update Tool
//
//	Finding available software
//	Software Update found the following new or updated software:
//	* Label: macOS Sonoma 14.5-23F79
//		Title: macOS Sonoma 14.5, Version: 14.5, Size: 7180348K, Recommended: YES, Action: restart,
//	* Label: Safari17.5MajorSU-17.5
//		Title: Safari, Version: 17.5, Size: 138648K, Recommended: YES,
//
// A `* Label:` line opens a new entry; the immediately following
// indented line carries comma-separated metadata. Fields not present
// in the metadata stay at their zero values.
func parseSoftwareUpdateList(stdout string) []parsedSoftwareUpdate {
	var out []parsedSoftwareUpdate
	var current *parsedSoftwareUpdate

	lines := strings.Split(stdout, "\n")
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		if rest, ok := stripPrefix(trimmed, "* Label:"); ok {
			if current != nil {
				out = append(out, *current)
			}
			current = &parsedSoftwareUpdate{label: strings.TrimSpace(rest)}
			continue
		}

		if current == nil {
			continue
		}

		// Metadata line lives at one indent level below `* Label:`. If
		// the trimmed line still starts with `* Label:` we'd have
		// caught it above, so anything else here is metadata.
		applySoftwareUpdateMetadata(trimmed, current)
	}
	if current != nil {
		out = append(out, *current)
	}
	return out
}

// applySoftwareUpdateMetadata parses one metadata line — comma-
// separated `Key: value` pairs — and writes the recognized fields onto
// the in-progress entry. Unknown keys (some macOS versions add `Build`
// or extra annotations) are quietly ignored.
func applySoftwareUpdateMetadata(line string, u *parsedSoftwareUpdate) {
	for _, field := range strings.Split(line, ", ") {
		field = strings.TrimRight(field, ",")
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		idx := strings.Index(field, ": ")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(field[:idx])
		value := strings.TrimSpace(field[idx+2:])

		switch key {
		case "Title":
			u.title = value
		case "Version":
			u.version = value
		case "Size":
			u.size = value
		case "Recommended":
			// macOS emits YES/NO; be tolerant of casing.
			u.recommended = strings.EqualFold(value, "YES")
		case "Action":
			u.action = value
		}
	}
}

// stripPrefix returns the substring after `prefix` plus true, or the
// original string plus false when the prefix is absent. Saves a
// `strings.HasPrefix` + `strings.TrimPrefix` pair at each call site.
func stripPrefix(s, prefix string) (string, bool) {
	if !strings.HasPrefix(s, prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

// =============================================================================
// shared plist helpers
// =============================================================================

// boolFromPlist coerces a plist key to bool. Through the plist.Decode
// helper, real plist booleans arrive as `bool`; integer 0/1 fallbacks
// (used historically by `defaults write -int`) arrive as `float64`
// after the helper's JSON round-trip. Both shapes count.
func boolFromPlist(d map[string]any, key string) bool {
	switch v := d[key].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int64:
		return v != 0
	case int:
		return v != 0
	case string:
		// MDM-managed preferences sometimes encode booleans as
		// "true"/"false" strings. Be tolerant.
		return strings.EqualFold(v, "true") || v == "1"
	}
	return false
}

func stringFromPlist(d map[string]any, key string) string {
	if v, ok := d[key].(string); ok {
		return v
	}
	return ""
}
