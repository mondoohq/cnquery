// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testutils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLinuxMock_ConcurrentAccess verifies that calling LinuxMock() from
// multiple goroutines does not cause a fatal crash.
//
// Before the fix, concurrent calls would panic with
// "fatal error: concurrent map iteration and map write" because Local()
// stored osSchema in the global extensibleSchema and then mutated it
// via osSchema.Add(coreSchema) without synchronization.
func TestLinuxMock_ConcurrentAccess(t *testing.T) {
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			rt := LinuxMock()
			assert.NotNil(t, rt)
		}()
	}

	wg.Wait()
}
