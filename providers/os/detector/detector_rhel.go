// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"bufio"
	"bytes"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/parsers"
)

const (
	modulesDir = "/etc/dnf/modules.d"
	// TODO: Is this the same on newer rhel versions? DNF?
	reposDir = "/etc/yum.repos.d"
)

type RhelModule struct {
	Name  string
	State string
}

func getActivatedRhelModules(conn shared.Connection) []string {
	afs := &afero.Afero{Fs: conn.FileSystem()}
	ok, err := afs.DirExists(modulesDir)
	if err != nil || !ok {
		return []string{}
	}

	files, err := afs.ReadDir(modulesDir)
	if err != nil {
		return []string{}
	}

	modules := []string{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".module") {
			continue
		}

		content, err := afs.ReadFile(filepath.Join(modulesDir, file.Name()))
		if err != nil {
			continue
		}

		module := RhelModule{}
		scanner := bufio.NewScanner(bytes.NewReader(content))
		for scanner.Scan() {
			s := strings.Split(scanner.Text(), "=")
			if len(s) != 2 {
				continue
			}

			switch strings.ToLower(s[0]) {
			case "name":
				module.Name = strings.TrimSpace(s[1])
			case "state":
				module.State = strings.TrimSpace(s[1])
			}
		}

		// We are only interested in enabled modules
		if module.State != "enabled" {
			continue
		}

		modules = append(modules, module.Name)
	}

	return modules
}

// getActivatedRhelSupportLevels returns the support level of the currently activated rhel repositories
// It currently detects the following support levels:
//   - eus (Extended Update Support)
//   - e4s (Enhanced Extendeded Update Support)
func getActivatedRhelSupportLevels(conn shared.Connection) []string {
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
			switch {
			case strings.Contains(section, "baseos-e4s-"):
				supportLevel = "e4s"
			case strings.Contains(section, "baseos-eus-"):
				supportLevel = "eus"
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
