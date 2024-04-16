// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/resources/packages"
)

func TestMacOsXPackageParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/packages_macos.toml", &inventory.Asset{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := packages.ParseMacOSPackages(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of packages")

	assert.Equal(t, "Preview", m[0].Name, "pkg name detected")
	assert.Equal(t, "10.0", m[0].Version, "pkg version detected")
	assert.Equal(t, packages.MacosPkgFormat, m[0].Format, "pkg format detected")
	assert.Equal(t, packages.PkgFilesIncluded, m[0].FilesAvailable)
	assert.Equal(t, []packages.FileRecord{{Path: "/Applications/Preview.app"}}, m[0].Files)

	assert.Equal(t, "Contacts", m[1].Name, "pkg name detected")
	assert.Equal(t, "11.0", m[1].Version, "pkg version detected")
	assert.Equal(t, packages.MacosPkgFormat, m[1].Format, "pkg format detected")
	assert.Equal(t, packages.PkgFilesIncluded, m[1].FilesAvailable)
	assert.Equal(t, []packages.FileRecord{{Path: "/Applications/Contacts.app"}}, m[1].Files)
}
