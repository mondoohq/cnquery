// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers/os/registry"
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

func defenderPolicyItems(values map[string]int64) map[string]registry.RegistryKeyItem {
	res := make(map[string]registry.RegistryKeyItem, len(values))
	for k, v := range values {
		res[strings.ToLower(k)] = registry.RegistryKeyItem{
			Key:   k,
			Value: registry.RegistryKeyValue{Kind: registry.DWORD, Number: v},
		}
	}
	return res
}

// Current Defender platforms report none of the LocalSettingOverride*
// preferences through Get-MpPreference, so each field must come from the
// Group Policy value WindowsDefender.admx defines for it.
func TestResolveLocalSettingOverrides(t *testing.T) {
	t.Run("not configured is null", func(t *testing.T) {
		o := resolveLocalSettingOverrides(&windows.MpPreference{}, &defenderPolicy{})
		assert.Equal(t, defenderOverrides{}, o)
	})

	t.Run("each field reads its own policy value", func(t *testing.T) {
		pol := &defenderPolicy{
			realTime: defenderPolicyItems(map[string]int64{
				"LocalSettingOverrideDisableRealtimeMonitoring": 1,
				"LocalSettingOverrideDisableBehaviorMonitoring": 0,
				"LocalSettingOverrideDisableIOAVProtection":     1,
				"LocalSettingOverrideDisableOnAccessProtection": 0,
			}),
			spynet: defenderPolicyItems(map[string]int64{"LocalSettingOverrideSpynetReporting": 1}),
			scan: defenderPolicyItems(map[string]int64{
				"LocalSettingOverrideScanParameters":   0,
				"LocalSettingOverrideScheduleDay":      1,
				"LocalSettingOverrideAvgCPULoadFactor": 0,
			}),
		}
		o := resolveLocalSettingOverrides(&windows.MpPreference{}, pol)

		check := func(name string, got *bool, want bool) {
			require.NotNil(t, got, name)
			assert.Equal(t, want, *got, name)
		}
		check("realtimeMonitoring", o.realtimeMonitoring, true)
		check("disableBehaviorMonitoring", o.disableBehaviorMonitoring, false)
		check("disableIOAVProtection", o.disableIOAVProtection, true)
		check("disableOnAccessProtection", o.disableOnAccessProtection, false)
		check("spynetReporting", o.spynetReporting, true)
		check("scanParameters", o.scanParameters, false)
		check("scanScheduleDay", o.scanScheduleDay, true)
		check("avgCPULoadFactor", o.avgCPULoadFactor, false)
		assert.Nil(t, o.disableIntrusionPreventionSystem, "Group Policy has no setting for this override")
	})

	t.Run("a value under the wrong subkey is not read", func(t *testing.T) {
		pol := &defenderPolicy{
			scan: defenderPolicyItems(map[string]int64{"LocalSettingOverrideDisableRealtimeMonitoring": 1}),
		}
		o := resolveLocalSettingOverrides(&windows.MpPreference{}, pol)
		assert.Nil(t, o.realtimeMonitoring)
	})

	t.Run("a reported preference wins over the policy", func(t *testing.T) {
		f := false
		pol := &defenderPolicy{
			realTime: defenderPolicyItems(map[string]int64{"LocalSettingOverrideDisableRealtimeMonitoring": 1}),
		}
		o := resolveLocalSettingOverrides(&windows.MpPreference{LocalSettingOverrideRealtimeMonitoring: &f}, pol)
		require.NotNil(t, o.realtimeMonitoring)
		assert.False(t, *o.realtimeMonitoring)
	})
}
