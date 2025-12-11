// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmdtest"
	"github.com/stretchr/testify/require"
)

var once sync.Once
var testDir string

func setup() {
	// build cnquery
	cmd := exec.Command("go", "build", "../../apps/cnquery/cnquery.go")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalf("building cnquery: %v", err)
	}

	// install local provider
	if err := exec.Command("bash", "-c", "cd ../.. && make providers/build/os providers/install/os").Run(); err != nil {
		log.Fatalf("building os provider: %v", err)
	}

	// create a fake directory to use for testing purposes (providers, config, etc.)
	dir, err := os.MkdirTemp("", "mondoo")
	if err != nil {
		log.Fatalf("creating directory: %v", err)
	}
	testDir = dir

	// provider install places the provider in the "$(HOME)/.config/mondoo/providers/${$@_NAME}") but we
	// want to test it in isolation. Therefore, we copy the provider to the current directory .providers
	osProviderPath := filepath.Join(testDir, "os")
	if err := os.MkdirAll(osProviderPath, 0755); err != nil {
		log.Fatalf("creating directory: %v", err)
	}

	distPath, err := filepath.Abs("../../providers/os/dist")
	if err != nil {
		log.Fatalf("unable to expand dist path: %v", err)
	}

	if err := os.CopyFS(osProviderPath, os.DirFS(distPath)); err != nil {
		log.Fatalf("copying provider: %v", err)
	}
}

func TestMain(m *testing.M) {
	ret := m.Run()
	os.Exit(ret)
}

func TestCompare(t *testing.T) {
	once.Do(setup)
	ts, err := cmdtest.Read("testdata")
	require.NoError(t, err)

	// Set a fake config path to avoid loading the real configuration
	// file from the system running this tests
	os.Setenv("MONDOO_CONFIG_PATH", path.Join(testDir, "foo"))
	// Override providers path with the fake test directory
	os.Setenv("PROVIDERS_PATH", testDir)
	// Disable auto-update to avoid installing providers
	os.Setenv("MONDOO_AUTO_UPDATE", "false")

	ts.DisableLogging = true
	ts.Commands["cnquery"] = cmdtest.Program("cnquery")
	ts.Run(t, false) // set to true to update test files
}
