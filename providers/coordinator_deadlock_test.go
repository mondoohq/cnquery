// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestCoordinatorSchemaLockOrder guards against an ABBA deadlock between
// coordinator.mutex and extensibleSchema.sync. Before LoadSchema stopped
// taking coordinator.mutex this test hung deterministically:
//
//	goroutine B: schema.Lookup(miss) ── holds schema.sync ──▶ unsafeLoadAll
//	             ──▶ coordinator.LoadSchema(name) ── wants coordinator.mutex
//	goroutine A: coordinator.GetRunningProvider(new) ── holds coordinator.mutex
//	             ──▶ unsafeStartProvider ──▶ schema.Add ── wants schema.sync
//
// In a scan this is a dispatcher worker compiling the first policy filter
// that references a not-yet-loaded resource (B) while the main goroutine
// connects a discovered child asset whose provider isn't running yet (A) —
// e.g. a github org scan starting the k8s provider for a manifest found in
// a repo. Once wedged, nothing times out; the job runs to its deadline.
func TestCoordinatorSchemaLockOrder(t *testing.T) {
	active, err := ListActive()
	if err != nil {
		t.Fatal(err)
	}
	if len(active) < 2 {
		t.Skip("needs at least two installed providers for unsafeLoadAll to iterate")
	}

	c := newCoordinator()
	// LoadSchema resolves names through c.providers; in production this is
	// populated by the first unsafeStartProvider. Populating it here makes
	// LoadSchema do the real (slow) LoadResources call under c.mutex.
	c.providers = active

	bDone := make(chan struct{})
	go func() {
		defer close(bDone)
		// Any miss with lastRefreshed < LastProviderInstall triggers unsafeLoadAll.
		c.schema.Lookup("resource.that.does.not.exist")
	}()

	// Let B enter unsafeLoadAll (it holds schema.sync from here until it has
	// called LoadSchema for every installed provider).
	time.Sleep(20 * time.Millisecond)

	aDone := make(chan struct{})
	go func() {
		defer close(aDone)
		// Builtin provider: no plugin subprocess, but the same lock order —
		// c.mutex held, then schema.Add wants schema.sync.
		_, _ = c.GetRunningProvider(BuiltinCoreID, UpdateProvidersConfig{})
	}()

	timeout := time.After(15 * time.Second)
	for _, done := range []chan struct{}{aDone, bDone} {
		select {
		case <-done:
		case <-timeout:
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			for _, g := range strings.Split(string(buf[:n]), "\n\n") {
				if strings.Contains(g, "extensibleSchema") || strings.Contains(g, "coordinator)") {
					t.Log(g)
				}
			}
			t.Fatal("deadlock: GetRunningProvider and schema.Lookup never returned")
		}
	}
}
