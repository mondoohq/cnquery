// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMpComputerStatus(t *testing.T) {
	data, err := os.ReadFile("testdata/defender_computer_status.json")
	require.NoError(t, err)

	status, err := ParseMpComputerStatus(data)
	require.NoError(t, err)

	assert.True(t, status.AMServiceEnabled)
	assert.True(t, status.AntivirusEnabled)
	assert.True(t, status.AntispywareEnabled)
	assert.True(t, status.RealTimeProtectionEnabled)
	assert.True(t, status.BehaviorMonitorEnabled)
	assert.True(t, status.IsTamperProtected)
	assert.Equal(t, "Signatures", status.TamperProtectionSource)
	assert.Equal(t, "1.405.319.0", status.AntivirusSignatureVersion)
	assert.Equal(t, "Off", status.SmartAppControlState)

	// time parsing of the "/Date(ms)/" form
	ts := DefenderTime(status.AntivirusSignatureLastUpdated)
	require.NotNil(t, ts)
	assert.Equal(t, int64(1705334400), ts.Unix())

	// null time fields resolve to nil
	assert.Nil(t, DefenderTime(status.FullScanStartTime))
	assert.Nil(t, DefenderTime(status.SmartAppControlExpiration))
}

func TestParseMpPreference(t *testing.T) {
	data, err := os.ReadFile("testdata/defender_preference.json")
	require.NoError(t, err)

	pref, err := ParseMpPreference(data)
	require.NoError(t, err)

	// scan
	assert.Equal(t, int64(1), pref.ScanParameters)
	assert.True(t, pref.DisableEmailScanning)
	assert.False(t, pref.DisableArchiveScanning)

	// cloud
	assert.Equal(t, int64(2), pref.MAPSReporting)
	assert.Equal(t, int64(1), pref.SubmitSamplesConsent)
	assert.Equal(t, int64(2), pref.CloudBlockLevel)

	// threat actions
	assert.Equal(t, int64(2), pref.SevereThreatDefaultAction)

	// per-threat-ID overrides: large unsigned ids preserved as strings
	require.Len(t, pref.ThreatIDDefaultAction_Ids, 2)
	assert.Equal(t, "2147519003", pref.ThreatIDDefaultAction_Ids[0])
	require.Len(t, pref.ThreatIDDefaultAction_Actions, 2)
	assert.Equal(t, int64(6), pref.ThreatIDDefaultAction_Actions[0])

	// controlled folder access
	assert.Equal(t, int64(1), pref.EnableControlledFolderAccess)
	require.Len(t, pref.ControlledFolderAccessProtectedFolders, 1)

	// network protection
	assert.Equal(t, int64(1), pref.EnableNetworkProtection)
	assert.True(t, pref.AllowNetworkProtectionOnWinServer)

	// exclusions: array form
	require.Len(t, pref.ExclusionPath, 2)
	assert.Equal(t, "C:\\Temp", pref.ExclusionPath[0])
	// exclusions: scalar form coerced into a single-element slice
	require.Len(t, pref.ExclusionExtension, 1)
	assert.Equal(t, ".log", pref.ExclusionExtension[0])
	// exclusions: null resolves to an empty slice
	assert.Empty(t, pref.ExclusionIpAddress)

	// ASR rules
	require.Len(t, pref.AttackSurfaceReductionRules_Ids, 2)
	assert.Equal(t, "56a863a9-875e-4185-98a7-b882c64b5ce5", pref.AttackSurfaceReductionRules_Ids[0])
	assert.Equal(t, int64(1), pref.AttackSurfaceReductionRules_Actions[0])
	assert.Equal(t,
		"Block abuse of exploited vulnerable signed drivers",
		AttackSurfaceReductionRuleNames[pref.AttackSurfaceReductionRules_Ids[0]],
	)

	// schedule time fields preserved as raw strings across PowerShell versions
	assert.Equal(t, "00:00:00", pref.ScanScheduleQuickScanTimeString())
	assert.NotEmpty(t, pref.ScanScheduleTimeString())

	// behavioral network blocks
	assert.Equal(t, int64(1), pref.BruteForceProtectionConfiguredState)
	assert.Equal(t, int64(2), pref.BruteForceProtectionAggressiveness)
	assert.Equal(t, int64(1), pref.RemoteEncryptionProtectionConfiguredState)

	// local setting overrides
	assert.False(t, pref.LocalSettingOverrideSpynetReporting)
	assert.False(t, pref.LocalSettingOverrideRealtimeMonitoring)

	// misc
	assert.Equal(t, int64(1), pref.PUAProtection)
	assert.True(t, pref.RandomizeScheduleTaskTimes)
	// the "RePorts" casing is matched via the explicit json tag
	assert.True(t, pref.DisableGenericReports)
}

func TestParseMpThreats(t *testing.T) {
	data, err := os.ReadFile("testdata/defender_threats.json")
	require.NoError(t, err)

	threats, err := ParseMpThreats(data)
	require.NoError(t, err)
	require.Len(t, threats, 2)

	assert.Equal(t, int64(2147519003), threats[0].ThreatID)
	assert.Equal(t, "Virus:DOS/EICAR_Test_File", threats[0].ThreatName)
	assert.False(t, threats[0].IsActive)
	require.Len(t, threats[0].Resources, 1)

	assert.True(t, threats[1].IsActive)
	assert.True(t, threats[1].DidThreatExecute)
	require.Len(t, threats[1].Resources, 2)
}

func TestParseMpThreats_Single(t *testing.T) {
	// PowerShell emits a bare object when there is exactly one result.
	single := `{"ThreatID":2147519003,"ThreatName":"Virus:DOS/EICAR_Test_File","SeverityID":5,"IsActive":false,"Resources":"file:_C:\\eicar.com"}`
	threats, err := ParseMpThreats([]byte(single))
	require.NoError(t, err)
	require.Len(t, threats, 1)
	assert.Equal(t, int64(2147519003), threats[0].ThreatID)
	require.Len(t, threats[0].Resources, 1)
}

func TestParseMpThreats_Empty(t *testing.T) {
	threats, err := ParseMpThreats([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, threats)
}

func TestIsDefenderUnavailable(t *testing.T) {
	assert.True(t, isDefenderUnavailable("The term 'Get-MpComputerStatus' is not recognized as the name of a cmdlet"))
	assert.True(t, isDefenderUnavailable("CommandNotFoundException"))
	assert.False(t, isDefenderUnavailable("Access is denied"))
}

// defenderUnavailable must work on non-English Windows. The localized "command
// not recognized" stderr text is not reliable, so the sentinel exit code set by
// the Get-Command guard is the primary, language-independent signal.
func TestDefenderUnavailable(t *testing.T) {
	t.Run("sentinel exit code, no stderr", func(t *testing.T) {
		// The Get-Command guard exits before emitting any (localized) text.
		assert.True(t, defenderUnavailable(defenderUnavailableExitCode, ""))
	})

	t.Run("sentinel exit code on a non-English host", func(t *testing.T) {
		// Worst case: fully localized prose with no English phrase and no cmdlet
		// token to fall back on. The heuristic alone cannot detect it, but the
		// sentinel exit code still flags it correctly.
		german := "Die Benennung wurde nicht als Name eines Cmdlets erkannt. Überprüfen Sie die Schreibweise des Namens."
		assert.False(t, isDefenderUnavailable(german), "heuristic cannot match fully localized text")
		assert.True(t, defenderUnavailable(defenderUnavailableExitCode, german))
	})

	t.Run("English stderr fallback without the sentinel", func(t *testing.T) {
		// Older path: a command-not-found surfaced via stderr on an English host.
		assert.True(t, defenderUnavailable(1, "The term 'Get-MpComputerStatus' is not recognized"))
	})

	t.Run("genuine error is not treated as unavailable", func(t *testing.T) {
		// A real failure (e.g. access denied) on a non-sentinel exit code must be
		// surfaced, not swallowed as "Defender not installed".
		assert.False(t, defenderUnavailable(1, "Zugriff verweigert"))
		assert.False(t, defenderUnavailable(1, "Access is denied"))
	})
}

// Every Defender script must carry the locale-independent availability guard so
// a missing cmdlet exits with the sentinel before any localized error text.
func TestDefenderScriptsHaveAvailabilityGuard(t *testing.T) {
	cases := map[string]struct {
		script string
		cmdlet string
	}{
		"computer status":  {defenderComputerStatusScript, "Get-MpComputerStatus"},
		"preferences":      {defenderPreferenceScript, "Get-MpPreference"},
		"threats":          {defenderThreatScript, "Get-MpThreat"},
		"threat detection": {defenderThreatDetectionScript, "Get-MpThreatDetection"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, tc.script, "Get-Command "+tc.cmdlet, "guard must probe the cmdlet")
			assert.Contains(t, tc.script, "exit 200", "guard must exit with the sentinel code")
			// The pipeline itself is still present after the guard.
			assert.Contains(t, tc.script, tc.cmdlet+" | ConvertTo-Json")
		})
	}
}
