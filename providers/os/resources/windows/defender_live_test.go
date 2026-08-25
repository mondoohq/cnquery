// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The defender_*_win<version>.json fixtures were captured with
// `Get-MpComputerStatus | ConvertTo-Json -Compress` and
// `Get-MpPreference | ConvertTo-Json -Compress -Depth 3` from live Windows
// Server 2016, 2019, 2022 and 2025 hosts, all running Defender platform
// 4.18.26070.9. Only the machine identifier was replaced with a synthetic
// value and the CIM metadata wrappers (CimClass, CimInstanceProperties,
// CimSystemProperties) that ConvertTo-Json emits alongside the properties were
// removed; every reported property is verbatim.
//
// Both cmdlets return the properties of a CIM instance, so their shapes come
// from the Defender platform rather than the Windows build. The platform
// updates itself independently of the OS, which is why all three versions
// agree.
var defenderLiveVersions = []string{"2016", "2019", "2022", "2025"}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return data
}

// TestMpComputerStatusLiveDecodes guards against the field types drifting away
// from what Defender actually returns. A single mismatched type aborts the
// whole json.Unmarshal, so one wrong tag takes down every field of
// windows.defender.status, not just its own.
func TestMpComputerStatusLiveDecodes(t *testing.T) {
	for _, v := range defenderLiveVersions {
		t.Run(v, func(t *testing.T) {
			status, err := ParseMpComputerStatus(readFixture(t, "defender_computer_status_win"+v+".json"))
			require.NoError(t, err)

			assert.True(t, status.AMServiceEnabled)
			assert.True(t, status.AntivirusEnabled)
			assert.True(t, status.AntispywareEnabled)
			assert.True(t, status.RealTimeProtectionEnabled)
			assert.True(t, status.BehaviorMonitorEnabled)
			assert.True(t, status.NISEnabled)
			assert.True(t, status.OnAccessProtectionEnabled)
			assert.True(t, status.IoavProtectionEnabled)
			assert.True(t, status.IsVirtualMachine)
			assert.False(t, status.IsTamperProtected)
			assert.False(t, status.RebootRequired)
			assert.Equal(t, "4.18.26070.9", status.AMProductVersion)
			assert.Equal(t, "Off", status.SmartAppControlState)

			// Device control reports names, not numbers. Defender leaves the
			// default enforcement unset on a host with no device-control
			// policy, so it must not read as a number either.
			assert.Equal(t, "Disabled", status.DeviceControlState)
			assert.Empty(t, status.DeviceControlDefaultEnforcement)

			// A host that has never run a scan reports uint32 max for the scan
			// age and leaves the scan times null.
			assert.Equal(t, int64(4294967295), status.FullScanAge)
			assert.Equal(t, int64(4294967295), status.QuickScanAge)
			assert.Nil(t, DefenderTime(status.FullScanStartTime))
			assert.Nil(t, DefenderTime(status.QuickScanEndTime))
			assert.Nil(t, DefenderTime(status.SmartAppControlExpiration))

			// Signature timestamps arrive in the PowerShell 5.1 "/Date(ms)/"
			// form and must parse.
			assert.NotNil(t, DefenderTime(status.AntivirusSignatureLastUpdated))
		})
	}
}

// TestMpPreferenceLiveDecodes is the Get-MpPreference counterpart.
func TestMpPreferenceLiveDecodes(t *testing.T) {
	for _, v := range defenderLiveVersions {
		t.Run(v, func(t *testing.T) {
			pref, err := ParseMpPreference(readFixture(t, "defender_preference_win"+v+".json"))
			require.NoError(t, err)

			// Defender reports this as a boolean, not as the 0/1/2 enum the
			// Set-MpPreference documentation once implied.
			assert.False(t, pref.CheckForSignaturesBeforeRunningScan)

			assert.Equal(t, int64(1), pref.ScanParameters)
			assert.Equal(t, int64(50), pref.ScanAvgCPULoadFactor)
			assert.Equal(t, int64(2), pref.MAPSReporting)
			assert.Equal(t, int64(1), pref.SubmitSamplesConsent)
			assert.Equal(t, int64(90), pref.QuarantinePurgeItemsAfterDelay)
			assert.Equal(t, "MicrosoftUpdateServer|MMPC", pref.SignatureFallbackOrder)
			assert.True(t, pref.EnableDnsSinkhole)
			assert.True(t, pref.DisableEmailScanning)
			assert.False(t, pref.DisableRealtimeMonitoring)

			// Nothing is configured on these hosts, so every list preference
			// comes back as a JSON null rather than an empty array.
			assert.Empty(t, pref.ExclusionPath)
			assert.Empty(t, pref.ExclusionExtension)
			assert.Empty(t, pref.ExclusionProcess)
			assert.Empty(t, pref.ExclusionIpAddress)
			assert.Empty(t, pref.AttackSurfaceReductionRules_Ids)
			assert.Empty(t, pref.AttackSurfaceReductionRules_Actions)
			assert.Empty(t, pref.ThreatIDDefaultAction_Ids)
			assert.Empty(t, pref.ThreatIDDefaultAction_Actions)
			assert.Empty(t, pref.ControlledFolderAccessAllowedApplications)
			assert.Empty(t, pref.ControlledFolderAccessProtectedFolders)
		})
	}
}

// TestDefenderFixturesAgreeAcrossVersions pins the observation that the
// property set is identical across all four Windows Server versions, so a
// future divergence shows up as a test failure with the differing names rather
// than as a field that quietly reads zero on one version.
func TestDefenderFixturesAgreeAcrossVersions(t *testing.T) {
	for _, prefix := range []string{"defender_computer_status_win", "defender_preference_win"} {
		t.Run(prefix, func(t *testing.T) {
			var base map[string]json.RawMessage
			for _, v := range defenderLiveVersions {
				var got map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(readFixture(t, prefix+v+".json"), &got))
				if base == nil {
					base = got
					continue
				}
				assert.ElementsMatch(t, keysOf(base), keysOf(got),
					fmt.Sprintf("%s%s reports a different property set", prefix, v))
			}
		})
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
