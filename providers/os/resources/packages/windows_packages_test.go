// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/registry"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

func TestWindowsAppPackagesParser(t *testing.T) {
	f, err := os.Open("./testdata/windows_packages.json")
	require.NoError(t, err)
	defer f.Close()

	pf := &inventory.Platform{
		Name:    "windows",
		Version: "10.0.18363",
		Arch:    "x86",
		Family:  []string{"windows"},
	}
	pkgs, err := ParseWindowsAppPackages(pf, f)
	assert.Nil(t, err)
	assert.Equal(t, 19, len(pkgs), "detected the right amount of packages")

	p := findPkg(pkgs, "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913")
	assert.Equal(t, Package{
		Name:    "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
		Version: "14.28.29913.0",
		Arch:    "x86",
		Format:  "windows/app",
		PUrl:    `pkg:windows/windows/Microsoft%20Visual%20C%2B%2B%202015-2019%20Redistributable%20%28x86%29%20-%2014.28.29913@14.28.29913.0?arch=x86`,
		CPEs: []string{
			"cpe:2.3:a:microsoft_corporation:microsoft_visual_c\\+\\+_2015-2019_redistributable_\\(x86\\)_-_14.28.29913:14.28.29913.0:*:*:*:*:*:*:*",
			"cpe:2.3:a:microsoft:microsoft_visual_c\\+\\+_2015-2019_redistributable_\\(x86\\)_-_14.28.29913:14.28.29913.0:*:*:*:*:*:*:*",
			"cpe:2.3:a:microsoft:microsoft_visual_c\\+\\+_2015-2019_redistributable_\\(x86\\)_-_14.28.29913:14.28.29913:*:*:*:*:*:*:*",
		},
		Vendor: "Microsoft Corporation",
	}, p)

	// check empty return
	pkgs, err = ParseWindowsAppxPackages(pf, strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(pkgs), "detected the right amount of packages")
}

func TestWindowsAppxPackagesParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/windows_2019.toml", &inventory.Asset{
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

	pf := &inventory.Platform{
		Name:    "windows",
		Version: "10.0.18363",
		Arch:    "x86",
		Family:  []string{"windows"},
	}

	pkgs, err := ParseWindowsAppxPackages(pf, c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 28, len(pkgs), "detected the right amount of packages")

	p := findPkg(pkgs, "Microsoft.Windows.Cortana")
	assert.Equal(t, Package{
		Name:    "Microsoft.Windows.Cortana",
		Version: "1.11.5.17763",
		Arch:    "neutral",
		Format:  "windows/appx",
		PUrl:    "pkg:appx/windows/Microsoft.Windows.Cortana@1.11.5.17763?arch=x86",
		// TODO: this is a bug in the CPE generation, we need to extract the publisher from the package
		CPEs: []string{
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.windows.cortana:1.11.5.17763:*:*:*:*:*:*:*",
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.windows.cortana:1.11.5:*:*:*:*:*:*:*",
		},
		Vendor: "CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US",
	}, p)

	// check empty return
	pkgs, err = ParseWindowsAppxPackages(pf, strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(pkgs), "detected the right amount of packages")
}

func TestWindowsHotFixParser(t *testing.T) {
	mock, err := mock.New(0, "./testdata/windows_2019.toml", &inventory.Asset{
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
	p := findPkg(pkgs, "KB4486553")
	assert.Equal(t, Package{
		Name:        "KB4486553",
		Description: "Update",
		Format:      "windows/hotfix",
	}, p)

	// check empty return
	hotfixes, err = ParseWindowsHotfixes(strings.NewReader(""))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(hotfixes), "detected the right amount of packages")
}

func TestGetPackageFromRegistryKeyItems(t *testing.T) {
	t.Run("get package from registry key items that are empty", func(t *testing.T) {
		items := []registry.RegistryKeyItem{}
		p := getPackageFromRegistryKeyItems(items, &inventory.Platform{
			Family: []string{"windows"},
		})
		assert.Nil(t, p)
	})
	t.Run("get package from registry key items with missing required values", func(t *testing.T) {
		items := []registry.RegistryKeyItem{
			{
				Key: "DisplayName",
				Value: registry.RegistryKeyValue{
					Kind:   registry.SZ,
					String: "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
				},
			},
		}
		p := getPackageFromRegistryKeyItems(items, &inventory.Platform{
			Family: []string{"windows"},
		})
		assert.Nil(t, p)
	})

	t.Run("get package from registry key items", func(t *testing.T) {
		items := []registry.RegistryKeyItem{
			{
				Key: "DisplayName",
				Value: registry.RegistryKeyValue{
					Kind:   registry.SZ,
					String: "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
				},
			},
			{
				Key: "UninstallString",
				Value: registry.RegistryKeyValue{
					Kind:   registry.SZ,
					String: "UninstallString",
				},
			},
			{
				Key: "DisplayVersion",
				Value: registry.RegistryKeyValue{
					Kind:   registry.SZ,
					String: "14.28.29913.0",
				},
			},
			{
				Key: "Publisher",
				Value: registry.RegistryKeyValue{
					Kind:   registry.SZ,
					String: "Microsoft Corporation",
				},
			},
		}
		p := getPackageFromRegistryKeyItems(items, &inventory.Platform{
			Name:   "windows",
			Arch:   "x86",
			Family: []string{"windows"},
		})
		CPEs, err := cpe.NewPackage2Cpe(
			"Microsoft Corporation",
			"Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
			"14.28.29913.0",
			"",
			"")

		assert.Nil(t, err)

		expected := &Package{
			Name:    "Microsoft Visual C++ 2015-2019 Redistributable (x86) - 14.28.29913",
			Version: "14.28.29913.0",
			Arch:    "x86",
			Format:  "windows/app",
			CPEs:    CPEs,
			Vendor:  "Microsoft Corporation",
			PUrl:    "pkg:windows/windows/Microsoft%20Visual%20C%2B%2B%202015-2019%20Redistributable%20%28x86%29%20-%2014.28.29913@14.28.29913.0?arch=x86",
		}
		assert.NotNil(t, p)
		assert.Equal(t, expected, p)
	})
}

func TestToPackage(t *testing.T) {
	winAppxPkg := winAppxPackages{
		Name:         "Microsoft.Windows.Cortana",
		Version:      "1.11.5.17763",
		Publisher:    "CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US",
		Architecture: 0,
	}

	pf := &inventory.Platform{
		Name:    "windows",
		Version: "10.0.18363",
		Arch:    "x86",
		Family:  []string{"windows"},
	}

	pkg := winAppxPkg.toPackage(pf)

	expected := Package{
		Name:    "Microsoft.Windows.Cortana",
		Version: "1.11.5.17763",
		Arch:    "x86",
		Format:  "windows/appx",
		PUrl:    "pkg:appx/windows/Microsoft.Windows.Cortana@1.11.5.17763?arch=x86",
		Vendor:  "CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US",
		CPEs: []string{
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.windows.cortana:1.11.5.17763:*:*:*:*:*:*:*",
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.windows.cortana:1.11.5:*:*:*:*:*:*:*",
		},
	}

	assert.Equal(t, expected, pkg)
}
