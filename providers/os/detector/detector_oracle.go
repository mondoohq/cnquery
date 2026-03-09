// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/parsers"
)

// getActivatedOracleSupportLevels returns the support level of the currently activated Oracle Linux repositories.
// It currently detects the following support levels:
//   - els (Extended Lifecycle Support)
//
// Oracle ELS repos have section names with an _ELS suffix, e.g. ol7_latest_ELS, ol7_UEKR6_ELS.
func getActivatedOracleSupportLevels(conn shared.Connection) []string {
	afs := &afero.Afero{Fs: conn.FileSystem()}
	ok, err := afs.DirExists(reposDir)
	if err != nil || !ok {
		return []string{}
	}

	files, err := afs.ReadDir(reposDir)
	if err != nil {
		return []string{}
	}

	supportLevels := []string{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".repo") {
			continue
		}

		content, err := afs.ReadFile(filepath.Join(reposDir, file.Name()))
		if err != nil {
			continue
		}

		repoIni := parsers.ParseIni(string(content), "=")
		if repoIni == nil {
			continue
		}

		for section, fields := range repoIni.Fields {
			supportLevel := ""
			if strings.HasSuffix(section, "_ELS") || strings.Contains(section, "_ELS/") {
				supportLevel = "els"
			}
			if supportLevel == "" {
				continue
			}
			if subFieldsMap, ok := fields.(map[string]interface{}); ok {
				if enabled, ok := subFieldsMap["enabled"]; ok {
					if v, ok := enabled.(string); ok && v == "1" {
						supportLevels = append(supportLevels, supportLevel)
					}
				}
			}
		}
	}

	slices.Sort(supportLevels)
	supportLevels = slices.Compact(supportLevels)

	return supportLevels
}
