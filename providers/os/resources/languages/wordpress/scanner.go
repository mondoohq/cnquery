// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package wordpress

import (
	"bufio"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// WordPressPlugin represents a parsed WordPress plugin.
type WordPressPlugin struct {
	// Slug is the plugin directory name (e.g., "akismet").
	Slug string
	// Version is from the "Stable tag" header.
	Version string
	// DisplayName is from the "=== Name ===" first line.
	DisplayName string
	// License is from the "License" header.
	License string
	// RequiresWp is from the "Requires at least" header.
	RequiresWp string
	// TestedUpTo is from the "Tested up to" header.
	TestedUpTo string
	// FilePath is the path to the readme.txt file.
	FilePath string
}

// ScanPluginDir scans a WordPress plugins directory for installed plugins.
// Each subdirectory containing a readme.txt is treated as a plugin.
func ScanPluginDir(afs *afero.Afero, dir string) ([]WordPressPlugin, error) {
	entries, err := afs.ReadDir(dir)
	if err != nil {
		log.Debug().Err(err).Str("path", dir).Msg("mql[wordpress]> could not read plugins directory")
		return nil, nil
	}

	var plugins []WordPressPlugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		readmePath := path.Join(dir, slug, "readme.txt")

		if exists, _ := afs.Exists(readmePath); !exists {
			continue
		}

		plugin, err := parseReadme(afs, readmePath, slug)
		if err != nil {
			log.Debug().Err(err).Str("path", readmePath).Msg("mql[wordpress]> could not parse readme.txt")
			continue
		}
		if plugin != nil {
			plugins = append(plugins, *plugin)
		}
	}

	return plugins, nil
}

// parseReadme reads a WordPress plugin readme.txt and extracts metadata headers.
func parseReadme(afs *afero.Afero, readmePath, slug string) (*WordPressPlugin, error) {
	f, err := afs.Open(readmePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	plugin := &WordPressPlugin{
		Slug:     slug,
		FilePath: readmePath,
	}

	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// First line: === Plugin Name ===
		if lineNum == 1 {
			if name := extractPluginName(line); name != "" {
				plugin.DisplayName = name
			}
			continue
		}

		// Stop at the description (empty line after headers or a line without a colon)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Headers are done once we hit an empty line after having parsed some
			if plugin.Version != "" {
				break
			}
			continue
		}

		// Parse "Key: Value" headers
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch strings.ToLower(key) {
		case "stable tag":
			plugin.Version = value
		case "license":
			plugin.License = value
		case "requires at least":
			plugin.RequiresWp = value
		case "tested up to":
			plugin.TestedUpTo = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Skip plugins without a version
	if plugin.Version == "" {
		return nil, nil
	}

	return plugin, nil
}

// extractPluginName extracts the name from "=== Plugin Name ===" format.
func extractPluginName(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "===") || !strings.HasSuffix(line, "===") {
		return ""
	}
	name := strings.TrimPrefix(line, "===")
	name = strings.TrimSuffix(name, "===")
	return strings.TrimSpace(name)
}
