// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/plist"
)

// Safari extension types to enumerate
var safariExtensionTypes = []string{
	"com.apple.Safari.web-extension",
	"com.apple.Safari.extension",
	"com.apple.Safari.content-blocker",
}

// Regex to extract extension path from pluginkit output
// Example line: "    Path = /Applications/Example.app/Contents/PlugIns/Extension.appex"
var pluginkitPathRegex = regexp.MustCompile(`^\s*Path\s*=\s*(.+)$`)

func (s *mqlSafari) extensions() ([]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	// Check if running on macOS
	pf := conn.Asset().Platform
	if pf == nil || pf.Family == nil || !slices.Contains(pf.Family, "darwin") {
		return nil, nil
	}

	// Check if pluginkit command exists
	afs := conn.FileSystem()
	if _, err := afs.Stat("/usr/bin/pluginkit"); err != nil {
		log.Warn().Msg("pluginkit command not found at /usr/bin/pluginkit, cannot enumerate Safari extensions")
		return []any{}, nil
	}

	seen := make(map[string]bool)
	var extensions []any

	for _, extType := range safariExtensionTypes {
		// Run pluginkit to list extensions of this type
		cmd, err := conn.RunCommand("pluginkit -mAvvv -p " + extType)
		if err != nil {
			return nil, err
		}

		if cmd.ExitStatus != 0 {
			stderr, _ := io.ReadAll(cmd.Stderr)
			return nil, fmt.Errorf("pluginkit failed for %s: %s", extType, string(stderr))
		}

		scanner := bufio.NewScanner(cmd.Stdout)
		for scanner.Scan() {
			line := scanner.Text()
			matches := pluginkitPathRegex.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}

			path := strings.TrimSpace(matches[1])
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true

			// Skip system extensions
			if strings.HasPrefix(path, "/System/") {
				continue
			}

			ext, err := newSafariExtension(s.MqlRuntime, conn, path, extType)
			if err != nil {
				// Skip extensions we can't parse
				continue
			}
			extensions = append(extensions, ext)
		}
	}

	return extensions, nil
}

func newSafariExtension(runtime *plugin.Runtime, conn shared.Connection, path string, extType string) (*mqlSafariExtension, error) {
	afs := conn.FileSystem()

	// Parse Info.plist from the extension bundle
	infoPlistPath := filepath.Join(path, "Contents", "Info.plist")
	f, err := afs.Open(infoPlistPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := plist.Decode(f)
	if err != nil {
		return nil, err
	}

	// Extract extension metadata
	identifier, _ := data.GetString("CFBundleIdentifier")
	name, _ := data.GetString("CFBundleName")
	if name == "" {
		name, _ = data.GetString("CFBundleDisplayName")
	}
	version, _ := data.GetString("CFBundleShortVersionString")
	if version == "" {
		version, _ = data.GetString("CFBundleVersion")
	}
	description, _ := data.GetString("NSHumanReadableCopyright")

	// Determine container app path and name
	containerAppPath := ""
	containerAppName := ""
	if idx := strings.Index(path, ".app/"); idx != -1 {
		containerAppPath = path[:idx+4] // Include ".app"
		containerAppName = filepath.Base(containerAppPath)
		containerAppName = strings.TrimSuffix(containerAppName, ".app")
	}

	// Derive extension type name from the pluginkit type
	extensionTypeName := strings.TrimPrefix(extType, "com.apple.Safari.")

	// Create the resource with a unique ID
	ext, err := CreateResource(runtime, "safari.extension", map[string]*llx.RawData{
		"__id":             llx.StringData(identifier + "|" + path),
		"identifier":       llx.StringData(identifier),
		"name":             llx.StringData(name),
		"version":          llx.StringData(version),
		"description":      llx.StringData(description),
		"extensionType":    llx.StringData(extensionTypeName),
		"path":             llx.StringData(path),
		"containerAppPath": llx.StringData(containerAppPath),
		"containerAppName": llx.StringData(containerAppName),
	})
	if err != nil {
		return nil, err
	}

	return ext.(*mqlSafariExtension), nil
}
