// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"encoding/xml"
	"io"
	"path"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

type SlesProduct struct {
	Summary  string        `xml:"summary"`
	Register RegisterEntry `xml:"register"`
}

type RegisterEntry struct {
	Target string `xml:"target"`
	Flavor string `xml:"flavor"`
}

func getActivatedSlesModules(conn shared.Connection) []string {
	afs := &afero.Afero{Fs: conn.FileSystem()}
	ok, err := afs.DirExists("/etc/products.d")
	if err != nil || !ok {
		return []string{}
	}

	files, err := afs.ReadDir("/etc/products.d")
	if err != nil {
		return []string{}
	}

	modules := []string{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".prod") {
			continue
		}

		content, err := afs.ReadFile("/etc/products.d/" + file.Name())
		if err != nil {
			continue
		}

		var product SlesProduct
		err = xml.Unmarshal(content, &product)
		if err != nil {
			continue
		}

		// We are only interested in modules and extensions
		if product.Register.Flavor != "module" && product.Register.Flavor != "extension" {
			continue
		}

		// We need to trim the prefix "SUSE " for some modules, to match the ecosystem
		// The same applies to the " Module" suffix
		moduleName := strings.TrimPrefix(product.Summary, "SUSE ")
		moduleName = strings.TrimSuffix(moduleName, " Module")
		modules = append(modules, moduleName)
	}

	return modules
}

func getSlesBaseProduct(conn shared.Connection) string {
	fs := conn.FileSystem()
	linkreader, ok := fs.(afero.LinkReader)
	var link string
	if ok {
		var err error
		link, err = linkreader.ReadlinkIfPossible("/etc/products.d/baseproduct")
		if err != nil || link == "" {
			return ""
		}
	} else {
		cmd, err := conn.RunCommand("readlink /etc/products.d/baseproduct")
		if err != nil || cmd.ExitStatus != 0 {
			return ""
		}
		lBytes, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return ""
		}
		link = strings.TrimSpace(string(lBytes))
	}

	// Get file name from the symlink
	name := path.Base(link)

	// trim the ".prod" suffix
	name = strings.TrimSuffix(name, ".prod")
	return strings.ToLower(name)
}
