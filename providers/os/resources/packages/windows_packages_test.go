// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/powershell"
)

func TestWindowsAppPackagesParser(t *testing.T) {
	f, err := os.Open("./testdata/windows_packages.json")
	require.NoError(t, err)
	defer f.Close()

	m, err := ParseWindowsAppPackages(f)
	assert.Nil(t, err)
	assert.Equal(t, 19, len(m), "detected the right amount of packages")

	p := Package{
		Name:    "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
		Version: "14.28.29913.0",
		Arch:    "",
		Format:  "windows/app",
	}
	assert.Contains(t, m, p)

	// check empty return
	m, err = ParseWindowsAppxPackages(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(m), "detected the right amount of packages")
}

func TestWindowsAppxPackagesParser(t *testing.T) {
	mock, err := mock.New("./testdata/windows_2019.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"windows"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(powershell.Wrap(WINDOWS_QUERY_APPX_PACKAGES))
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseWindowsAppxPackages(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 28, len(m), "detected the right amount of packages")

	p := Package{
		Name:    "Microsoft.Windows.Cortana",
		Version: "1.11.5.17763",
		Arch:    "neutral",
		Format:  "windows/appx",
	}
	assert.Contains(t, m, p)

	// check empty return
	m, err = ParseWindowsAppxPackages(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(m), "detected the right amount of packages")
}

func TestWindowsHotFixParser(t *testing.T) {
	mock, err := mock.New("./testdata/windows_2019.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"windows"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := mock.RunCommand(powershell.Wrap(WINDOWS_QUERY_HOTFIXES))
	if err != nil {
		t.Fatal(err)
	}

	hotfixes, err := ParseWindowsHotfixes(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(hotfixes), "detected the right amount of packages")

	timestamp := hotfixes[0].InstalledOnTime()
	assert.NotNil(t, timestamp)

	pkgs := HotFixesToPackages(hotfixes)
	p := Package{
		Name:        "KB4486553",
		Description: "Update",
		Format:      "windows/hotfix",
	}
	assert.Contains(t, pkgs, p)

	// check empty return
	hotfixes, err = ParseWindowsHotfixes(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(hotfixes), "detected the right amount of packages")
}
