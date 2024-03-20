// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/test"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var once sync.Once

// setup builds cnquery locally
func setup() {
	// build cnspec
	if err := exec.Command("go", "build", "../../apps/cnquery/cnquery.go").Run(); err != nil {
		log.Fatalf("building cnquery: %v", err)
	}

	// install local provider
	if err := exec.Command("bash", "-c", "cd ../.. && make providers/build/os providers/install/os").Run(); err != nil {
		log.Fatalf("building os provider: %v", err)
	}

	providersPATH := os.Getenv("PROVIDERS_PATH")
	if providersPATH != "" {

		// provider install places the provider in the "$(HOME)/.config/mondoo/providers/${$@_NAME}") but we
		// want to test it in isolation. Therefore, we copy the provider to the current directory .providers
		osProviderPath := filepath.Join(providersPATH, "os")
		if err := os.MkdirAll(osProviderPath, 0755); err != nil {
			log.Fatalf("creating directory: %v", err)
		}

		if err := exec.Command("cp", "-r", os.ExpandEnv("../../providers/os/dist"), osProviderPath).Run(); err != nil {
			log.Fatalf("copying provider: %v", err)
		}
	}
}

const mqlPackagesQuery = "packages"

type mqlPackages []struct {
	Packages []struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"packages.list,omitempty"`
}

const mqlPlatformQuery = "asset.platform"

type mqlPlatform []struct {
	Platform string `json:"asset.platform,omitempty"`
}

type connections []struct {
	name   string
	binary string
	args   []string
	tests  []mqlTest
}

type mqlTest struct {
	query    string
	expected func(*testing.T, test.Runner)
}

func TestOsProviderSharedTests(t *testing.T) {
	once.Do(setup)

	connections := connections{
		{
			name:   "local",
			binary: "./cnquery",
			args:   []string{"run", "local"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.True(t, len(x.Platform) > 0)
					},
				},
			},
		},
		{
			name:   "fs",
			binary: "./cnquery",
			args:   []string{"run", "fs", "--path", "./testdata/fs"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.Equal(t, "debian", x.Platform)
					},
				},
			},
		},
		{
			name:   "docker",
			binary: "./cnquery",
			args:   []string{"run", "docker", "alpine:latest"},
			tests: []mqlTest{
				{
					mqlPackagesQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPackages
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.NotNil(t, x.Packages)
						assert.True(t, len(x.Packages) > 0)
					},
				},
				{
					mqlPlatformQuery,
					func(t *testing.T, r test.Runner) {
						var c mqlPlatform
						err := r.Json(&c)
						assert.NoError(t, err)

						x := c[0]
						assert.Equal(t, "alpine", x.Platform)
					},
				},
			},
		},
	}

	// iterate over all tests for all connections
	for _, cc := range connections {
		for _, tt := range cc.tests {

			t.Run(cc.name+"/"+tt.query, func(t *testing.T) {
				r := test.NewCliTestRunner(cc.binary, append(cc.args, "-c", tt.query, "-j")...)
				err := r.Run()
				require.NoError(t, err)
				assert.Equal(t, 0, r.ExitCode())
				assert.NotNil(t, r.Stdout())
				assert.NotNil(t, r.Stderr())

				tt.expected(t, r)
			})
		}
	}
}
