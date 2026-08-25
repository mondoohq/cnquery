// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceGuardScriptFitsCommandLine(t *testing.T) {
	assert.LessOrEqual(t, len(PSGetDeviceGuard), PSMaxScriptLength,
		"the Device Guard collection script has grown past what fits on the command line")
}

func TestParseDeviceGuardStatus(t *testing.T) {
	f, err := os.Open("./testdata/deviceguard_status.json")
	require.NoError(t, err)
	defer f.Close()

	dg, err := ParseDeviceGuardStatus(f)
	require.NoError(t, err)

	// Each tag is pinned individually: a mistyped one yields a zero value, and
	// a zero value here reads as "not enforced" or "nothing running" on a host
	// that is neither.
	assert.Equal(t, PSInt64Array{1, 2, 3, 5, 7}, dg.AvailableSecurityProperties)
	assert.Equal(t, PSInt64Array{1, 2}, dg.SecurityServicesConfigured)
	assert.Equal(t, PSInt64Array{2}, dg.SecurityServicesRunning)
	require.NotNil(t, dg.CodeIntegrityPolicyEnforcementStatus)
	assert.Equal(t, int64(2), *dg.CodeIntegrityPolicyEnforcementStatus)
	require.NotNil(t, dg.UsermodeCodeIntegrityPolicyEnforcementStatus)
	assert.Equal(t, int64(0), *dg.UsermodeCodeIntegrityPolicyEnforcementStatus)
	require.NotNil(t, dg.VirtualizationBasedSecurityStatus)
	assert.Equal(t, int64(2), *dg.VirtualizationBasedSecurityStatus)
}

func TestDeviceGuardConfiguredIsNotRunning(t *testing.T) {
	// The reading this whole change exists for: Credential Guard (service 1)
	// is configured and did not start, which the policy registry values alone
	// cannot express.
	f, err := os.Open("./testdata/deviceguard_status.json")
	require.NoError(t, err)
	defer f.Close()

	dg, err := ParseDeviceGuardStatus(f)
	require.NoError(t, err)
	assert.Contains(t, dg.SecurityServicesConfigured, int64(1))
	assert.NotContains(t, dg.SecurityServicesRunning, int64(1))
}

func TestParseDeviceGuardStatusNullables(t *testing.T) {
	t.Run("absent scalars stay null rather than becoming 0", func(t *testing.T) {
		// 0 means "off" for every one of these, so decoding an absent value to
		// 0 would report a definite reading the host never gave.
		dg, err := ParseDeviceGuardStatus(strings.NewReader(
			`{"AvailableSecurityProperties":null,"CodeIntegrityPolicyEnforcementStatus":null,` +
				`"UsermodeCodeIntegrityPolicyEnforcementStatus":null,"VirtualizationBasedSecurityStatus":null}`))
		require.NoError(t, err)
		assert.Nil(t, dg.AvailableSecurityProperties)
		assert.Nil(t, dg.SecurityServicesConfigured)
		assert.Nil(t, dg.SecurityServicesRunning)
		assert.Nil(t, dg.CodeIntegrityPolicyEnforcementStatus)
		assert.Nil(t, dg.UsermodeCodeIntegrityPolicyEnforcementStatus)
		assert.Nil(t, dg.VirtualizationBasedSecurityStatus)
	})

	t.Run("an explicit 0 is not absent", func(t *testing.T) {
		dg, err := ParseDeviceGuardStatus(strings.NewReader(
			`{"SecurityServicesRunning":[],"VirtualizationBasedSecurityStatus":0}`))
		require.NoError(t, err)
		require.NotNil(t, dg.SecurityServicesRunning)
		assert.Len(t, dg.SecurityServicesRunning, 0)
		require.NotNil(t, dg.VirtualizationBasedSecurityStatus)
		assert.Equal(t, int64(0), *dg.VirtualizationBasedSecurityStatus)
	})

	t.Run("a wrapped list still decodes", func(t *testing.T) {
		dg, err := ParseDeviceGuardStatus(strings.NewReader(
			`{"SecurityServicesRunning":{"value":[1,2],"Count":2}}`))
		require.NoError(t, err)
		assert.Equal(t, PSInt64Array{1, 2}, dg.SecurityServicesRunning)
	})
}

func TestParseDeviceGuardStatusRejectsEmptyOutput(t *testing.T) {
	// No output means the reading cannot be trusted. Returning an all-null
	// status instead would let "nothing is running" stand as an answer.
	_, err := ParseDeviceGuardStatus(strings.NewReader("   \n"))
	assert.Error(t, err)
}
