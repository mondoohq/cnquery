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
	require.NoError(t, err)
	require.Equal(t, 29, len(pkgs), "detected the right amount of packages")

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

	p = findPkg(pkgs, "Microsoft.MicrosoftEdge.Stable")
	assert.Equal(t, Package{
		Name:    "Microsoft.MicrosoftEdge.Stable",
		Version: "112.0.1722.39",
		Arch:    "neutral",
		Format:  "windows/appx",
		PUrl:    "pkg:appx/windows/Microsoft.MicrosoftEdge.Stable@112.0.1722.39?arch=x86",
		// TODO: this is a bug in the CPE generation, we need to extract the publisher from the package
		CPEs: []string{
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.microsoftedge.stable:112.0.1722.39:*:*:*:*:*:*:*",
			"cpe:2.3:a:cn\\=microsoft_corporation\\,_o\\=microsoft_corporation\\,_l\\=redmond\\,_s\\=washington\\,_c\\=us:microsoft.microsoftedge.stable:112.0.1722:*:*:*:*:*:*:*",
		},
		Vendor: "CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US",
		Files: []FileRecord{
			{
				Path: "C:\\Program Files\\WindowsApps\\Microsoft.MicrosoftEdge.Stable_112.0.1722.39_neutral__8wekyb3d8bbwe",
			},
		},
		FilesAvailable: PkgFilesIncluded,
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

func findPkgByName(pkgs []Package, name string) *Package {
	for i := range pkgs {
		if pkgs[i].Name == name {
			return &pkgs[i]
		}
	}
	return nil
}

func TestFindAndUpdateMsSqlHotfixes(t *testing.T) {
	// Setup: create a list of packages with SQL Server hotfixes and SQL Server packages
	packages := []Package{
		{Name: "SQL Server 2019 Database Engine Services", Version: "15.0.2000.5", PUrl: "pkg:windows/windows/SQL%20Server%202019%20Database%20Engine%20Services@15.0.2000.5?arch=x86"},
		{Name: "SQL Server 2019 Shared Management Objects", Version: "15.0.2000.5", PUrl: "pkg:windows/windows/SQL%20Server%202019%20Shared%20Management%20Objects@15.0.2000.5?arch=x86"},
		// We should not update the setup package
		{Name: "Microsoft SQL Server 2019 Setup (English)", Version: "15.0.2123.5", PUrl: "pkg:windows/windows/Microsoft%20SQL%20Server%202019%20Setup%20%28English%29@15.0.2123.5?arch=x86"},
		{Name: "Hotfix KB5001090 SQL Server", Version: "15.0.4102.2", PUrl: "pkg:windows/windows/Hotfix%20KB5001090%20SQL%20Server@15.0.4102.2?arch=x86"},
		{Name: "Hotfix KB5001091 SQL Server", Version: "15.0.4123.1", PUrl: "pkg:windows/windows/Hotfix%20KB5001091%20SQL%20Server@15.0.4123.1?arch=x86"},
		{Name: "Not a hotfix", Version: "1.0.0", PUrl: "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86"},
	}

	// Step 1: Find SQL Server hotfixes
	hotfixes := findMsSqlHotfixes(packages)
	require.Len(t, hotfixes, 2, "expected 2 hotfixes")

	// Step 2: Get the latest hotfix (should be the last one after sorting)
	latestHotfix := hotfixes[len(hotfixes)-1]
	expectedLatestVersion := "15.0.4123.1"
	require.Equal(t, expectedLatestVersion, latestHotfix.Version, "expected latest hotfix version")

	// Step 3: Update SQL Server packages with the latest hotfix version
	updated := updateMsSqlPackages(packages, latestHotfix)

	// Step 4: Check that all SQL Server packages have the updated version
	pkg := findPkgByName(updated, "SQL Server 2019 Database Engine Services")
	require.NotNil(t, pkg, "SQL Server 2019 Database Engine Services package should exist")
	require.Equal(t, expectedLatestVersion, pkg.Version, "expected SQL Server 2019 Database Engine Services to have updated version")
	assert.Equal(t, "pkg:windows/windows/SQL%20Server%202019%20Database%20Engine%20Services@15.0.4123.1?arch=x86", pkg.PUrl)

	pkg = findPkgByName(updated, "SQL Server 2019 Shared Management Objects")
	require.NotNil(t, pkg, "SQL Server 2019 Shared Management Objects package should exist")
	require.Equal(t, expectedLatestVersion, pkg.Version, "expected SQL Server 2019 Shared Management Objects to have updated version")
	assert.Equal(t, "pkg:windows/windows/SQL%20Server%202019%20Shared%20Management%20Objects@15.0.4123.1?arch=x86", pkg.PUrl)

	pkg = findPkgByName(updated, "Microsoft SQL Server 2019 Setup (English)")
	require.NotNil(t, pkg, "Microsoft SQL Server 2019 Setup (English) package should exist")
	require.Equal(t, "15.0.2123.5", pkg.Version, "expected Microsoft SQL Server 2019 Setup (English) to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Microsoft%20SQL%20Server%202019%20Setup%20%28English%29@15.0.2123.5?arch=x86", pkg.PUrl)

	pkg = findPkgByName(updated, "Hotfix KB5001090 SQL Server")
	require.NotNil(t, pkg, "Hotfix KB5001090 SQL Server package should exist")
	require.Equal(t, "15.0.4102.2", pkg.Version, "expected Hotfix KB5001090 SQL Server to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Hotfix%20KB5001090%20SQL%20Server@15.0.4102.2?arch=x86", pkg.PUrl)

	pkg = findPkgByName(updated, "Hotfix KB5001091 SQL Server")
	require.NotNil(t, pkg, "Hotfix KB5001091 SQL Server package should exist")
	require.Equal(t, "15.0.4123.1", pkg.Version, "expected Hotfix KB5001091 SQL Server to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Hotfix%20KB5001091%20SQL%20Server@15.0.4123.1?arch=x86", pkg.PUrl)

	// Step 5: Ensure non-SQL Server packages are unchanged
	pkg = findPkgByName(updated, "Not a hotfix")
	require.NotNil(t, pkg, "Not a hotfix package should exist")
	require.Equal(t, "1.0.0", pkg.Version, "expected non-SQL Server package to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86", pkg.PUrl)
}

func TestFindAndUpdateMsSqlGDR_en(t *testing.T) {
	// Setup: create a list of packages with SQL Server hotfixes and SQL Server packages
	packages := []Package{
		{Name: "SQL Server 2022 Database Engine Services", Version: "16.0.1000.6", PUrl: "pkg:windows/windows/SQL%20Server%202022%20Database%20Engine%20Services@16.0.1000.6?arch=x86"},
		{Name: "SQL Server 2022 Shared Management Objects", Version: "16.0.1000.6", PUrl: "pkg:windows/windows/SQL%20Server%202022%20Shared%20Management%20Objects@16.0.1000.6?arch=x86"},
		// We should not update the setup package
		{Name: "Microsoft SQL Server 2022 Setup (English)", Version: "16.0.1000.6", PUrl: "pkg:windows/windows/Microsoft%20SQL%20Server%202022%20Setup%20%28English%29@16.0.1000.6?arch=x86"},
		{Name: "GDR 1115 for SQL Server 2022 (KB5035432) (64-bit)", Version: "16.0.1115.1", PUrl: "pkg:windows/windows/GDR%201115%20for%20SQL%20Server%202022%20%28KB5035432%29%20%2864-bit%29@16.0.1115.1?arch=x86"},
		{Name: "GDR 1105 for SQL Server 2022 (KB5029379) (64-bit)", Version: "16.0.1105.1", PUrl: "pkg:windows/windows/GDR%201105%20for%20SQL%20Server%202022%20%28KB5029379%29%20%2864-bit%29@16.0.1105.1?arch=x86"},
		{Name: "Not a hotfix", Version: "1.0.0", PUrl: "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86"},
	}

	// Step 1: Find SQL Server gdrUpdates
	gdrUpdates := findMsSqlGdrUpdates(packages)
	require.Len(t, gdrUpdates, 2, "expected 2 updates")

	// Step 2: Get the latest hotfix (should be the last one after sorting)
	latestUpdate := gdrUpdates[len(gdrUpdates)-1]
	expectedLatestVersion := "16.0.1115.1"
	require.Equal(t, expectedLatestVersion, latestUpdate.Version, "expected latest update version")

	// Step 3: Update SQL Server packages with the latest hotfix version
	packages = updateMsSqlPackages(packages, latestUpdate)

	// Step 4: Check that all SQL Server packages have the updated version
	pkg := findPkgByName(packages, "SQL Server 2022 Database Engine Services")
	require.NotNil(t, pkg, "SQL Server 2022 Database Engine Services package should exist")
	require.Equal(t, expectedLatestVersion, pkg.Version, "expected SQL Server 2022 Database Engine Services to have updated version")
	assert.Equal(t, "pkg:windows/windows/SQL%20Server%202022%20Database%20Engine%20Services@16.0.1115.1?arch=x86", pkg.PUrl)

	pkg = findPkgByName(packages, "GDR 1105 for SQL Server 2022 (KB5029379) (64-bit)")
	require.NotNil(t, pkg, "KB5029379 SQL Server package should exist")
	require.Equal(t, "16.0.1105.1", pkg.Version, "expected Hotfix KB5029379 SQL Server to remain unchanged")

	pkg = findPkgByName(packages, "GDR 1115 for SQL Server 2022 (KB5035432) (64-bit)")
	require.NotNil(t, pkg, "KB5035432 SQL Server package should exist")
	require.Equal(t, "16.0.1115.1", pkg.Version, "expected Hotfix KB5035432 SQL Server to remain unchanged")

	// Step 5: Ensure non-SQL Server packages are unchanged
	pkg = findPkgByName(packages, "Not a hotfix")
	require.NotNil(t, pkg, "Not a hotfix package should exist")
	require.Equal(t, "1.0.0", pkg.Version, "expected non-SQL Server package to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86", pkg.PUrl)
}

func TestFindAndUpdateMsSqlGDR_de(t *testing.T) {
	// Setup: create a list of packages with SQL Server hotfixes and SQL Server packages
	packages := []Package{
		{Name: "SQL Server 2022 Database Engine Services", Version: "16.0.1050.5", PUrl: "pkg:windows/windows/SQL%20Server%202022%20Database%20Engine%20Services@16.0.1050.5?arch=x86"},
		{Name: "SQL Server 2022 Shared Management Objects", Version: "16.0.1050.5", PUrl: "pkg:windows/windows/SQL%20Server%202022%20Shared%20Management%20Objects@16.0.1050.5?arch=x86"},
		// We should not update the setup package
		{Name: "Microsoft SQL Server 2022 Setup (English)", Version: "16.0.1050.5", PUrl: "pkg:windows/windows/Microsoft%20SQL%20Server%202022%20Setup%20%28English%29@16.0.1050.5?arch=x86"},
		{Name: "GDR 1115 für SQL Server 2022 (KB5035432) (64-bit)", Version: "16.0.1115.1", PUrl: "pkg:windows/windows/GDR%201115%20f%C3%BCr%20SQL%20Server%202022%20%28KB5035432%29%20%2864-bit%29@16.0.1115.1?arch=x86"},
		{Name: "GDR 1110 für SQL Server 2022 (KB5032968) (64-bit)", Version: "16.0.1110.1", PUrl: "pkg:windows/windows/GDR%201110%20f%C3%BCr%20SQL%20Server%202022%20%28KB5032968%29%20%2864-bit%29@16.0.1110.1?arch=x86"},
		{Name: "Not a hotfix", Version: "1.0.0", PUrl: "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86"},
	}

	// Step 1: Find SQL Server gdrUpdates
	gdrUpdates := findMsSqlGdrUpdates(packages)
	require.Len(t, gdrUpdates, 2, "expected 2 updates")

	// Step 2: Get the latest hotfix (should be the last one after sorting)
	latestUpdate := gdrUpdates[len(gdrUpdates)-1]
	expectedLatestVersion := "16.0.1115.1"
	require.Equal(t, expectedLatestVersion, latestUpdate.Version, "expected latest update version")

	// Step 3: Update SQL Server packages with the latest hotfix version
	updated := updateMsSqlPackages(packages, latestUpdate)

	// Step 4: Check that all SQL Server packages have the updated version
	pkg := findPkgByName(updated, "SQL Server 2022 Database Engine Services")
	require.NotNil(t, pkg, "SQL Server 2022 Database Engine Services package should exist")
	require.Equal(t, expectedLatestVersion, pkg.Version, "expected SQL Server 2022 Database Engine Services to have updated version")
	assert.Equal(t, "pkg:windows/windows/SQL%20Server%202022%20Database%20Engine%20Services@16.0.1115.1?arch=x86", pkg.PUrl)

	pkg = findPkgByName(updated, "GDR 1115 für SQL Server 2022 (KB5035432) (64-bit)")
	require.NotNil(t, pkg, "KB5035432 SQL Server package should exist")
	require.Equal(t, "16.0.1115.1", pkg.Version, "expected Hotfix KB5001090 SQL Server to remain unchanged")

	pkg = findPkgByName(updated, "GDR 1110 für SQL Server 2022 (KB5032968) (64-bit)")
	require.NotNil(t, pkg, "KB5032968 SQL Server package should exist")
	require.Equal(t, "16.0.1110.1", pkg.Version, "expected Hotfix KB5001091 SQL Server to remain unchanged")

	// Step 5: Ensure non-SQL Server packages are unchanged
	pkg = findPkgByName(updated, "Not a hotfix")
	require.NotNil(t, pkg, "Not a hotfix package should exist")
	require.Equal(t, "1.0.0", pkg.Version, "expected non-SQL Server package to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86", pkg.PUrl)
}

func TestFindAndUpdateMsSqlGDR_de_special_characters(t *testing.T) {
	// Setup: create a list of packages with SQL Server hotfixes and SQL Server packages
	packages := []Package{
		{Name: "SQL Server 2017 Database Engine Services", Version: "14.0.1000.169", PUrl: "pkg:windows/windows/SQL%20Server%202017%20Database%20Engine%20Services@14.0.1000.169?arch=x86"},
		{Name: "SQL Server 2017 Shared Management Objects", Version: "14.0.1000.169", PUrl: "pkg:windows/windows/SQL%20Server%202022%20Shared%20Management%20Objects@14.0.1000.169?arch=x86"},
		// We should not update the setup package
		{Name: "Microsoft SQL Server 2017 Setup (English)", Version: "14.0.1000.169", PUrl: "pkg:windows/windows/Microsoft%20SQL%20Server%202017%20Setup%20%28English%29@14.0.1050.5"},
		// This package contains a non breaking space between SQL Server and 2017
		{Name: "GDR 2042 für SQL Server 2017 (KB5014354) (64-bit)", Version: "14.0.2042.3", PUrl: "pkg:windows/windows/GDR%202042%20f%C3%BCr%20SQL%20Server%202017%20%28KB5014354%29%20%2864-bit%29@14.0.2042.3?arch=x86"},
		// This package contains a non breaking space between SQL Server and 2017
		{Name: "GDR 2037 für SQL Server 2017 (KB4583456) (64-bit)", Version: "14.0.2037.2", PUrl: "pkg:windows/windows/GDR%202037%20f%C3%BCr%20SQL%20Server%202017%20%28KB4583456%29%20%2864-bit%29@14.0.2037.2?arch=x86"},
		{Name: "Not a hotfix", Version: "1.0.0", PUrl: "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86"},
	}

	// Step 1: Find SQL Server gdrUpdates
	gdrUpdates := findMsSqlGdrUpdates(packages)
	require.Len(t, gdrUpdates, 2, "expected 2 updates")

	// Step 2: Get the latest hotfix (should be the last one after sorting)
	latestUpdate := gdrUpdates[len(gdrUpdates)-1]
	expectedLatestVersion := "14.0.2042.3"
	require.Equal(t, expectedLatestVersion, latestUpdate.Version, "expected latest update version")

	// Step 3: Update SQL Server packages with the latest hotfix version
	updated := updateMsSqlPackages(packages, latestUpdate)

	// Step 4: Check that all SQL Server packages have the updated version
	pkg := findPkgByName(updated, "SQL Server 2017 Database Engine Services")
	require.NotNil(t, pkg, "SQL Server 2017 Database Engine Services package should exist")
	require.Equal(t, expectedLatestVersion, pkg.Version, "expected SQL Server 2017 Database Engine Services to have updated version")
	assert.Equal(t, "pkg:windows/windows/SQL%20Server%202017%20Database%20Engine%20Services@14.0.2042.3?arch=x86", pkg.PUrl)

	// Step 5: Ensure non-SQL Server packages are unchanged
	pkg = findPkgByName(updated, "Not a hotfix")
	require.NotNil(t, pkg, "Not a hotfix package should exist")
	require.Equal(t, "1.0.0", pkg.Version, "expected non-SQL Server package to remain unchanged")
	assert.Equal(t, "pkg:windows/windows/Not%20a%20hotfix@1.0.0?arch=x86", pkg.PUrl)
}

func TestCreatePackage(t *testing.T) {
	t.Run("create package with non-breaking space in name", func(t *testing.T) {
		// The name contains a non-breaking space between Server and 2017
		pkg := createPackage("GDR 2042 für SQL Server 2017 (KB5014354) (64-bit)", "1234", "windows/app", "x86_64", "Microsoft", "", nil)
		require.NotNil(t, pkg, "expected package to be created")

		// Here we check that the name is replaced with a regular space
		assert.Equal(t, "GDR 2042 für SQL Server 2017 (KB5014354) (64-bit)", pkg.Name)
	})

	t.Run("create package with non-breaking space in name - unicode", func(t *testing.T) {
		// The name contains a non-breaking space between Server and 2017
		pkg := createPackage("GDR 2042 für SQL Server\u00a02017 (KB5014354) (64-bit)", "1234", "windows/app", "x86_64", "Microsoft", "", nil)
		require.NotNil(t, pkg, "expected package to be created")

		// Here we check that the name is replaced with a regular space
		assert.Equal(t, "GDR 2042 für SQL Server 2017 (KB5014354) (64-bit)", pkg.Name)
	})
}
