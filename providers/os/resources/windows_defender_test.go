// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/resources/windows"
)

// windows.defender.status and windows.defender.preferences are both field paths
// on windows.defender and registered resource names in their own right. Without
// an Init the dotted form instantiates the sub-resource, the parent's accessor
// never runs, and every field reads null - so a check reading "antivirus is
// enabled" off windows.defender.status.antivirusEnabled reports null on a host
// where Defender is running and protecting the machine.
func TestWindowsDefenderSingletonsAreReachableByTheirOwnPath(t *testing.T) {
	for _, path := range []string{"windows.defender.status", "windows.defender.preferences"} {
		t.Run(path, func(t *testing.T) {
			_, isField := getDataFields[path]
			require.True(t, isField, "%s should be a field path on windows.defender", path)

			factory, isResource := resourceFactories[path]
			require.True(t, isResource, "%s should also be a registered resource name", path)

			assert.NotNil(t, factory.Init,
				"%s resolves to the resource, not the field, so without an Init every field reads null", path)
		})
	}
}

// seedPrefs installs an already-fetched Get-MpPreference result onto a resource
// that CreateResource may have returned from the runtime cache, where another
// goroutine can already be reading it. This asserts what seedPrefs itself
// guarantees: concurrent seeds and reads are race-free, a reader never observes
// fetched as true with a nil prefs, and a second seed does not replace a result
// already handed out.
//
// It does not reproduce the unlocked-write bug this replaced. That write lived
// in preferences(), which needs a live runtime and connection to reach, so the
// race there is established by reading the code rather than by this test.
func TestWindowsDefenderPreferencesSeedIsSynchronized(t *testing.T) {
	const workers = 16

	p := &mqlWindowsDefenderPreferences{}
	seeded := &windows.MpPreference{ScanParameters: 2}

	// A start barrier, so the seeders and the readers actually overlap rather
	// than running one after another.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			p.seedPrefs(seeded)
		}()
		go func() {
			defer wg.Done()
			<-start
			prefs, err, ok := p.peekPrefs()
			if ok {
				assert.NoError(t, err)
				assert.NotNil(t, prefs, "fetched was observed true with a nil prefs")
			}
		}()
	}
	close(start)
	wg.Wait()

	prefs, err := p.getPrefs()
	require.NoError(t, err)
	require.NotNil(t, prefs)
	assert.Equal(t, int64(2), prefs.ScanParameters)

	// A later seed must not replace what has already been handed out.
	p.seedPrefs(&windows.MpPreference{ScanParameters: 99})
	prefs, err = p.getPrefs()
	require.NoError(t, err)
	assert.Equal(t, int64(2), prefs.ScanParameters)
}
