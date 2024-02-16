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

func TestPackageJson(t *testing.T) {
	tests := []struct {
		Fixture  string
		Expected packageJson
	}{
		{
			Fixture: "./testdata/package-json/author.json",
			Expected: packageJson{
				Name: "author.js",
				Author: &packageJsonPeople{
					Name:  "Barney Rubble",
					Email: "b@rubble.com",
					URL:   "http://barnyrubble.tumblr.com/",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/author_shorten.json",
			Expected: packageJson{
				Name: "author-shorten.js",
				Author: &packageJsonPeople{
					Name:  "Barney Rubble",
					Email: "b@rubble.com",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/author_shorten_url.json",
			Expected: packageJson{
				Name: "author-shorten.js",
				Author: &packageJsonPeople{
					Name:  "Barney Rubble",
					Email: "b@rubble.com",
					URL:   "http://barnyrubble.tumblr.com/",
				},
			},
		},
		{
			Fixture:  "./testdata/package-json/bugs.json",
			Expected: packageJson{
				// we do not parse bugs, so it should be empty
			},
		},
		{
			Fixture: "./testdata/package-json/bundle_dependencies.json",
			Expected: packageJson{
				Name:    "awesome-web-framework",
				Version: "1.0.0",
				// we do not parse bundle dependencies, so it should be empty
			},
		},
		{
			Fixture: "./testdata/package-json/contributors.json",
			Expected: packageJson{
				Contributors: []packageJsonPeople{
					{
						Name:  "Barney Rubble",
						Email: "b@rubble.com",
						URL:   "http://barnyrubble.tumblr.com/",
					},
				},
			},
		},
		{
			Fixture: "./testdata/package-json/contributors_shorten.json",
			Expected: packageJson{
				Contributors: []packageJsonPeople{
					{
						Name:  "Barney Rubble",
						Email: "b@rubble.com",
						URL:   "http://barnyrubble.tumblr.com/",
					},
				},
			},
		},
		{
			Fixture: "./testdata/package-json/cpu_exclude.json",
			Expected: packageJson{
				CPU: []string{"!arm", "!mips"},
			},
		},
		{
			Fixture: "./testdata/package-json/cpu_include.json",
			Expected: packageJson{
				CPU: []string{"x64", "ia32"},
			},
		},
		{
			Fixture: "./testdata/package-json/dependencies.json",
			Expected: packageJson{
				Dependencies: map[string]string{
					"foo": "1.0.0 - 2.9999.9999",
					"bar": ">=1.0.2 <2.1.2",
					"baz": ">1.0.2 <=2.3.4",
					"boo": "2.0.1",
					"qux": "<1.0.0 || >=2.3.1 <2.4.5 || >=2.5.2 <3.0.0",
					"asd": "http://asdf.com/asdf.tar.gz",
					"til": "~1.2",
					"elf": "~1.2.3",
					"two": "2.x",
					"thr": "3.3.x",
					"lat": "latest",
					"dyl": "file:../dyl",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/dev_dependencies.json",
			Expected: packageJson{
				Name:        "ethopia-waza",
				Description: "a delightfully fruity coffee varietal",
				Version:     "1.2.3",
				DevDependencies: map[string]string{
					"coffee-script": "~1.6.3",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/engines.json",
			Expected: packageJson{
				Engines: map[string]string{
					"node": ">=0.10.3 <15",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/homepage.json",
			Expected: packageJson{
				Homepage: "https://github.com/owner/project#readme",
			},
		},
		{
			Fixture: "./testdata/package-json/license_deprecated_01.json",
			Expected: packageJson{
				// we ignore those licenses for now
				License: &packageJsonLicense{
					Value: "",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/license_deprecated_02.json",
			Expected: packageJson{
				// we ignore those licenses for now
				License: nil,
			},
		},
		{
			Fixture: "./testdata/package-json/license_spdx.json",
			Expected: packageJson{
				License: &packageJsonLicense{
					Value: "BSD-3-Clause",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/license_spdx_expression.json",
			Expected: packageJson{
				License: &packageJsonLicense{
					Value: "(MIT OR Apache-2.0)",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/os_exclude.json",
			Expected: packageJson{
				OS: []string{"!win32"},
			},
		},
		{
			Fixture: "./testdata/package-json/os_include.json",
			Expected: packageJson{
				OS: []string{"darwin", "linux"},
			},
		},
		{
			Fixture: "./testdata/package-json/peer_dependencies.json",
			Expected: packageJson{
				Name:    "tea-latte",
				Version: "1.3.5",
				// we do not parse peerDependencies so it should be empty
			},
		},
		{
			Fixture: "./testdata/package-json/peer_dependencies_meta.json",
			Expected: packageJson{
				Name:    "tea-latte",
				Version: "1.3.5",
				// we do not parse peerDependenciesMeta so it should be empty
			},
		},
		{
			Fixture: "./testdata/package-json/private.json",
			Expected: packageJson{
				Private: true,
			},
		},
		{
			Fixture: "./testdata/package-json/repository.json",
			Expected: packageJson{
				Repository: packageJsonRepository{
					Type: "git",
					URL:  "https://github.com/npm/cli.git",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/repository_bitbucket.json",
			Expected: packageJson{
				Repository: packageJsonRepository{
					Type: "bitbucket",
					URL:  "user/repo",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/repository_directory.json",
			Expected: packageJson{
				Repository: packageJsonRepository{
					Type:      "git",
					URL:       "https://github.com/facebook/react.git",
					Directory: "packages/react-dom",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/repository_gh.json",
			Expected: packageJson{
				Repository: packageJsonRepository{
					Type: "github",
					URL:  "npm/npm",
				},
			},
		},
		{
			Fixture: "./testdata/package-json/workspaces.json",
			Expected: packageJson{
				Name: "workspace-example",
				// we do not parse workspaces so it should be empty
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.Fixture, func(t *testing.T) {
			f, err := os.Open(tests[i].Fixture)
			require.NoError(t, err)

			pkg := packageJson{}
			err = json.NewDecoder(f).Decode(&pkg)
			require.NoError(t, err)
			assert.Equal(t, tests[i].Expected, pkg)
		})
	}
}

func TestPackageJsonParser(t *testing.T) {
	f, err := os.Open("./testdata/package-json/express-package.json")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&PackageJsonParser{}).Parse(f, "path/package.json")
	assert.Nil(t, err)

	root := info.Root()
	assert.Equal(t, &Package{
		Name:              "express",
		Version:           "4.16.4",
		Purl:              "pkg:npm/express@4.16.4",
		Cpes:              []string{"cpe:2.3:a:express:express:4.16.4:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/package.json"},
	}, root)

	transitive := info.Transitive()
	assert.Equal(t, 30, len(transitive))
	p := findPkg(transitive, "path-to-regexp")
	assert.Equal(t, &Package{
		Name:              "path-to-regexp",
		Version:           "0.1.7",
		Purl:              "pkg:npm/path-to-regexp@0.1.7",
		Cpes:              []string{"cpe:2.3:a:path-to-regexp:path-to-regexp:0.1.7:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/package.json"},
	}, p)

	p = findPkg(transitive, "range-parser")
	assert.Equal(t, &Package{
		Name:              "range-parser",
		Version:           "~1.2.0",
		Purl:              "pkg:npm/range-parser@1.2.0",
		Cpes:              []string{"cpe:2.3:a:range-parser:range-parser:1.2.0:*:*:*:*:*:*:*"},
		EvidenceLocations: []string{"path/package.json"},
	}, p)
}
