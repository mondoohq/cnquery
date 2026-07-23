// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packagelockjson

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashesFor(t *testing.T) {
	// A single sha512 SRI decodes to a SHA-512 hash with a lower-case hex value.
	got := hashesFor("sha512-wLH6RzYPQAryrsJakc9I3k0aFWE/cJyWoUD8dQy186jxwtLgeQdVc0+NegNyab7MIPi7Hsv9A3hx6lM1rPH94A==")
	require.Len(t, got, 1)
	assert.Equal(t, "SHA-512", got[0].Alg)
	assert.Equal(t, "c0b1fa47360f400af2aec25a91cf48de4d1a15613f709c96a140fc750cb5f3a8f1c2d2e0790755734f8d7a037269becc20f8bb1ecbfd037871ea5335acf1fde0", got[0].Value)

	// Multiple space-separated entries yield one hash each, unknown algs skipped.
	multi := hashesFor("sha1-qBk8lPTdW3wLpjSFMv40C0OT13c= whirlpool-notanalg")
	require.Len(t, multi, 1)
	assert.Equal(t, "SHA-1", multi[0].Alg)

	// Empty, malformed (no dash), and undecodable base64 all yield nothing.
	assert.Nil(t, hashesFor(""))
	assert.Nil(t, hashesFor("   "))
	assert.Nil(t, hashesFor("sha512"))
	assert.Nil(t, hashesFor("sha512-!!!not base64!!!"))
}

func TestPackageLock(t *testing.T) {
	tests := []struct {
		Fixture  string
		Expected packageLock
	}{
		{
			Fixture: "testdata/lockfile-v0.json",
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
			Fixture: "testdata/lockfile-v1.json",
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
			Fixture: "testdata/lockfile-v2.json",
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
						Dev:       true,
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
			Fixture: "testdata/lockfile-v2-licenses.json",
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
			Fixture: "testdata/lockfile-v3.json",
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
			Fixture: "testdata/simple-lock.json",
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
