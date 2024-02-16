// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageLock(t *testing.T) {
	tests := []struct {
		Fixture  string
		Expected packageLock
	}{
		{
			Fixture: "testdata/package-lock/lockfile-v0.json",
			Expected: packageLock{
				Name:            "react-build",
				Version:         "15.1.0",
				LockfileVersion: 0,
				Requires:        false,
				Packages:        nil,
				Dependencies: map[string]packageLockDependency{
					"art": {
						Version:  "0.10.1",
						Resolved: "https://registry.npmjs.org/art/-/art-0.10.1.tgz",
					},
					"babel-cli": {
						Version:  "6.10.1",
						Resolved: "https://registry.npmjs.org/babel-cli/-/babel-cli-6.10.1.tgz",
					},
				},
			},
		},
		{
			Fixture: "testdata/package-lock/lockfile-v1.json",
			Expected: packageLock{
				Name:            "npm",
				Version:         "6.0.0",
				LockfileVersion: 1,
				Requires:        true,
				Dependencies: map[string]packageLockDependency{
					"JSONStream": {
						Version:   "1.3.2",
						Resolved:  "https://registry.npmjs.org/JSONStream/-/JSONStream-1.3.2.tgz",
						Integrity: "sha1-wQI3G27Dp887hHygDCC7D85Mbeo=",
					},
				},
			},
		},
		{
			Fixture: "testdata/package-lock/lockfile-v2.json",
			Expected: packageLock{
				Name:            "npm",
				Version:         "7.0.0",
				LockfileVersion: 2,
				Requires:        true,
				Packages: map[string]packageLockPackage{
					"": {
						Name:    "npm",
						Version: "7.0.0",
						License: packageLockLicense(
							[]string{"Artistic-2.0"},
						),
						Dependencies: map[string]string{
							"@npmcli/arborist":  "^1.0.0",
							"@npmcli/ci-detect": "^1.2.0",
						},
					},
					"node_modules/@babel/code-frame": {
						Version:   "7.10.4",
						Resolved:  "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.10.4.tgz",
						Integrity: "sha512-vG6SvB6oYEhvgisZNFRmRCUkLz11c7rp+tbNTynGqc6mS1d5ATd/sGyV6W0KZZnXRKMTzZDRgQT3Ou9jhpAfUg==",
						Dependencies: map[string]string{
							"@babel/highlight": "^7.10.4",
						},
					},
				},
				Dependencies: map[string]packageLockDependency{
					"@babel/code-frame": {
						Version:   "7.10.4",
						Resolved:  "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.10.4.tgz",
						Integrity: "sha512-vG6SvB6oYEhvgisZNFRmRCUkLz11c7rp+tbNTynGqc6mS1d5ATd/sGyV6W0KZZnXRKMTzZDRgQT3Ou9jhpAfUg==",
						Dev:       true,
					},
				},
			},
		},
		{
			Fixture: "testdata/package-lock/lockfile-v2-licenses.json",
			Expected: packageLock{
				Name:            "my-package",
				Version:         "1.0.0",
				LockfileVersion: 2,
				Requires:        true,
				Packages: map[string]packageLockPackage{
					"": {
						Name:    "my-package",
						Version: "1.0.0",
						License: packageLockLicense(
							[]string{"MIT", "Apache2"},
						),
					},
				},
			},
		},
		{
			Fixture: "testdata/package-lock/lockfile-v3.json",
			Expected: packageLock{
				Name:            "npm",
				Version:         "10.4.0",
				LockfileVersion: 3,
				Requires:        true,
				Packages: map[string]packageLockPackage{
					"": {
						Name:    "npm",
						Version: "10.4.0",
						License: packageLockLicense(
							[]string{"Artistic-2.0"},
						),
						Dependencies: map[string]string{
							"@isaacs/string-locale-compare": "^1.1.0",
						},
					},
					"node_modules/@isaacs/string-locale-compare": {
						Version:   "1.1.0",
						Resolved:  "https://registry.npmjs.org/@isaacs/string-locale-compare/-/string-locale-compare-1.1.0.tgz",
						Integrity: "sha512-SQ7Kzhh9+D+ZW9MA0zkYv3VXhIDNx+LzM6EJ+/65I3QY+enU6Itte7E5XX7EWrqLW2FN4n06GWzBnPoC3th2aQ==",
					},
				},
			},
		},
		{
			Fixture: "testdata/package-lock/simple-lock.json",
			Expected: packageLock{
				Name:            "simple",
				Version:         "1.0.0",
				LockfileVersion: 1,
				Requires:        true,
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.Fixture, func(t *testing.T) {
			f, err := os.Open(tests[i].Fixture)
			require.NoError(t, err)

			pkg := packageLock{}
			err = json.NewDecoder(f).Decode(&pkg)
			require.NoError(t, err)
			assert.Equal(t, tests[i].Expected, pkg)
		})
	}
}

func TestPackageJsonLockWithPackages(t *testing.T) {
	f, err := os.Open("./testdata/package-lock/lockfile-v2.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&PackageLockParser{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &Package{
		Name:              "npm",
		Version:           "7.0.0",
		Purl:              "pkg:npm/npm@7.0.0",
		Cpes:              []string{"cpe:2.3:a:npm:npm:7.0.0:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/to/package-lock.json"},
	}, root)

	transitive := info.Transitive()
	assert.Equal(t, 2, len(transitive))

	p := findPkg(transitive, "@babel/code-frame")
	assert.Equal(t, &Package{
		Name:              "@babel/code-frame",
		Version:           "7.10.4",
		Purl:              "pkg:npm/node-modules/%40babel@7.10.4",
		Cpes:              []string{"cpe:2.3:a:node_modules\\/\\@babel\\/code-frame:node_modules\\/\\@babel\\/code-frame:7.10.4:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/to/package-lock.json"},
	}, p)

}

func TestPackageJsonLockWithDependencies(t *testing.T) {
	f, err := os.Open("./testdata/package-lock/workbox-package-lock.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&PackageLockParser{}).Parse(f, "path/to/package-lock.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &Package{
		Name:              "workbox",
		Version:           "0.0.0",
		Purl:              "pkg:npm/workbox@0.0.0",
		Cpes:              []string{"cpe:2.3:a:workbox:workbox:0.0.0:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/to/package-lock.json"},
	}, root)

	transitive := info.Transitive()
	assert.Equal(t, 1299, len(transitive))

	p := findPkg(transitive, "@babel/generator")
	assert.Equal(t, &Package{
		Name:              "@babel/generator",
		Version:           "7.0.0",
		Purl:              "pkg:npm/%40babel/generator@7.0.0",
		Cpes:              []string{"cpe:2.3:a:\\@babel\\/generator:\\@babel\\/generator:7.0.0:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/to/package-lock.json"},
	}, p)

	p = findPkg(transitive, "@lerna/changed")
	assert.Equal(t, &Package{
		Name:              "@lerna/changed",
		Version:           "3.3.2",
		Purl:              "pkg:npm/%40lerna/changed@3.3.2",
		Cpes:              []string{"cpe:2.3:a:\\@lerna\\/changed:\\@lerna\\/changed:3.3.2:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/to/package-lock.json"},
	}, p)
}
