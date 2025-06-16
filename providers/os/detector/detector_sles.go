// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"encoding/xml"
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
