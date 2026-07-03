// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
)

// newHclTestRuntime stands up a plugin runtime over an HCL directory, wired
// exactly like the provider service does (same create/new/get/set hooks), so
// resource inits and generated Get* accessors behave as they do in a scan.
func newHclTestRuntime(t *testing.T, path string) *plugin.Runtime {
	t.Helper()
	asset := &inventory.Asset{
		Connections: []*inventory.Config{{
			Options: map[string]string{"path": path},
		}},
	}
	conn, err := connection.NewHclConnection(0, asset)
	require.NoError(t, err)
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

// writeLargeHclFixture generates a single-module config with enough blocks
// that refreshCache's populate loop has a meaningful execution window.
func writeLargeHclFixture(t *testing.T, resources, variables, outputs int) string {
	t.Helper()
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("provider \"google\" {\n  project = \"test\"\n}\n\n")
	for i := 0; i < resources; i++ {
		fmt.Fprintf(&sb, "resource \"google_storage_bucket\" \"b%d\" {\n  name = \"bucket-%d\"\n  location = \"EU\"\n}\n\n", i, i)
	}
	for i := 0; i < variables; i++ {
		fmt.Fprintf(&sb, "variable \"v%d\" {\n  default = %d\n}\n\n", i, i)
	}
	for i := 0; i < outputs; i++ {
		fmt.Fprintf(&sb, "output \"o%d\" {\n  value = var.v%d\n}\n\n", i, i)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(sb.String()), 0o644))
	return dir
}

// TestConcurrentTerraformResources mimics what a policy scan does on first
// touch of a terraform asset: many filter queries concurrently evaluating
// terraform.resources against a cold cache, then reading the block
// collections. Every worker must observe the same, complete collections.
//
// Run with -race: on the unfixed code the detector flags the concurrent
// (re-)initialization of registry-shared terraform.block instances
// (newMqlHclBlock), refreshCache reading blocks another goroutine is still
// writing, and the unsynchronized read of the internal resource list in
// initTerraformResources.
//
// The accessor phase runs after the build fan-out completed: the very first
// touch of a TValue cell under concurrency is unsynchronized in the SDK
// itself (GetOrCompute's fast path reads, and the runtime registry's
// get-then-set can hand out duplicate instances whose cells are published
// while another goroutine cold-reads them). That hazard cannot be fixed from
// provider code and is deliberately out of this test's scope.
func TestConcurrentTerraformResources(t *testing.T) {
	const (
		nResources = 300
		nVariables = 50
		nOutputs   = 50
		workers    = 64
	)
	dir := writeLargeHclFixture(t, nResources, nVariables, nOutputs)
	runtime := newHclTestRuntime(t, dir)

	type observation struct {
		resources int
		providers int
		variables int
	}
	obs := make([]observation, workers)

	// Phase 1: concurrent cold builds. This is the path every
	// `terraform.resources...` filter query takes (snapshot at resource
	// init); it parses, creates the shared block instances, and classifies
	// them.
	var start, done sync.WaitGroup
	start.Add(1)
	done.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer done.Done()
			start.Wait()

			args, _, err := initTerraformResources(runtime, map[string]*llx.RawData{})
			if err != nil {
				t.Error(err)
				return
			}
			obs[i].resources = len(args["list"].Value.([]any))
		}(i)
	}
	start.Done()
	done.Wait()

	// Phase 2: concurrent collection reads via the generated accessors,
	// against the now-built cache.
	var readsDone sync.WaitGroup
	readsDone.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer readsDone.Done()

			tfraw, err := CreateResource(runtime, "terraform", map[string]*llx.RawData{})
			if err != nil {
				t.Error(err)
				return
			}
			tf := tfraw.(*mqlTerraform)
			if p := tf.GetProviders(); p.Error == nil {
				obs[i].providers = len(p.Data)
			}
			if v := tf.GetVariables(); v.Error == nil {
				obs[i].variables = len(v.Data)
			}
		}(i)
	}
	readsDone.Wait()

	for i := 0; i < workers; i++ {
		require.Equalf(t, nResources, obs[i].resources, "worker %d saw a partial terraform.resources list", i)
		require.Equalf(t, 1, obs[i].providers, "worker %d saw a partial terraform.providers list", i)
		require.Equalf(t, nVariables, obs[i].variables, "worker %d saw a partial terraform.variables list", i)
	}
}
