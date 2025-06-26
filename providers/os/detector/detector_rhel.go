// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	modulesDir = "/etc/dnf/modules.d"
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

		content, err := afs.ReadFile(modulesDir + "/" + file.Name())
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
