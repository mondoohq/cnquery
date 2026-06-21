// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/registry"
)

// regItems builds a name->item map mirroring readRegistryKey output (keys are
// lower-cased) from a set of REG_DWORD name/value pairs.
func regItems(values map[string]int64) map[string]registry.RegistryKeyItem {
	items := map[string]registry.RegistryKeyItem{}
	for name, v := range values {
		items[lower(name)] = registry.RegistryKeyItem{
			Key:   name,
			Value: registry.RegistryKeyValue{Number: v},
		}
	}
	return items
}

func TestComputeWindowsUpdatePolicy(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		v := computeWindowsUpdatePolicy(nil, nil)
		assert.False(t, v.automaticUpdatesEnabled)
		assert.Nil(t, v.noAutoUpdate)
		assert.Nil(t, v.scheduledInstallDay)
		assert.Nil(t, v.deferQualityUpdatesPeriodInDays)
		assert.Nil(t, v.disablePauseUXAccess)
	})

	t.Run("automatic updates enabled (NoAutoUpdate=0)", func(t *testing.T) {
		au := regItems(map[string]int64{"NoAutoUpdate": 0, "ScheduledInstallDay": 0})
		v := computeWindowsUpdatePolicy(nil, au)
		assert.True(t, v.automaticUpdatesEnabled)
		require.NotNil(t, v.noAutoUpdate)
		assert.Equal(t, int64(0), *v.noAutoUpdate)
		// ScheduledInstallDay=0 ("every day") must be distinguishable from absent.
		require.NotNil(t, v.scheduledInstallDay)
		assert.Equal(t, int64(0), *v.scheduledInstallDay)
	})

	t.Run("automatic updates disabled (NoAutoUpdate=1)", func(t *testing.T) {
		au := regItems(map[string]int64{"NoAutoUpdate": 1})
		v := computeWindowsUpdatePolicy(nil, au)
		assert.False(t, v.automaticUpdatesEnabled)
		require.NotNil(t, v.noAutoUpdate)
		assert.Equal(t, int64(1), *v.noAutoUpdate)
	})

	t.Run("case-insensitive value names", func(t *testing.T) {
		// readRegistryKey lower-cases names; ensure lookups still resolve.
		au := map[string]registry.RegistryKeyItem{
			"noautoupdate": {Key: "noautoupdate", Value: registry.RegistryKeyValue{Number: 0}},
		}
		v := computeWindowsUpdatePolicy(nil, au)
		assert.True(t, v.automaticUpdatesEnabled)
	})

	t.Run("windows update for business deferrals", func(t *testing.T) {
		policy := regItems(map[string]int64{
			"ManagePreviewBuildsPolicyValue":         0,
			"DeferFeatureUpdates":                    1,
			"DeferFeatureUpdatesPeriodInDays":        180,
			"DeferQualityUpdates":                    1,
			"DeferQualityUpdatesPeriodInDays":        0,
			"AllowTemporaryEnterpriseFeatureControl": 0,
			"SetAllowOptionalContent":                0,
			"SetDisablePauseUXAccess":                1,
		})
		au := regItems(map[string]int64{"NoAutoRebootWithLoggedOnUsers": 0})
		v := computeWindowsUpdatePolicy(policy, au)

		require.NotNil(t, v.managePreviewBuilds)
		assert.Equal(t, int64(0), *v.managePreviewBuilds)
		require.NotNil(t, v.deferFeatureUpdatesPeriodInDays)
		assert.Equal(t, int64(180), *v.deferFeatureUpdatesPeriodInDays)
		// 0-day quality deferral must not collapse to "not configured".
		require.NotNil(t, v.deferQualityUpdatesPeriodInDays)
		assert.Equal(t, int64(0), *v.deferQualityUpdatesPeriodInDays)
		require.NotNil(t, v.disablePauseUXAccess)
		assert.Equal(t, int64(1), *v.disablePauseUXAccess)
		require.NotNil(t, v.noAutoRebootWithLoggedOnUsers)
		assert.Equal(t, int64(0), *v.noAutoRebootWithLoggedOnUsers)
	})
}

func TestDeriveCatalogSource(t *testing.T) {
	tests := []struct {
		name           string
		read           bool
		useWUServer    bool
		wsusServerURL  string
		auOptions      int64
		hasPolicyState bool
		want           string
	}{
		{name: "registry unreadable", read: false, want: "unknown"},
		{name: "automatic updates disabled", read: true, auOptions: 1, want: "disabled"},
		{name: "wsus managed", read: true, useWUServer: true, wsusServerURL: "http://wsus.local:8530", auOptions: 4, want: "wsus"},
		{name: "wsus url without UseWUServer falls through", read: true, useWUServer: false, wsusServerURL: "http://wsus.local:8530", auOptions: 4, want: "windowsUpdate"},
		{name: "windows update for business", read: true, hasPolicyState: true, auOptions: 0, want: "windowsUpdateForBusiness"},
		{name: "direct windows update", read: true, auOptions: 4, want: "windowsUpdate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveCatalogSource(tt.read, tt.useWUServer, tt.wsusServerURL, tt.auOptions, tt.hasPolicyState)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseWULastSuccessTime(t *testing.T) {
	got := parseWULastSuccessTime("2024-01-15 10:30:00")
	require.NotNil(t, got)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), got.UTC())

	assert.Nil(t, parseWULastSuccessTime(""))
	assert.Nil(t, parseWULastSuccessTime("not a timestamp"))
}

func TestFormatWULastError(t *testing.T) {
	assert.Equal(t, "", formatWULastError(0))
	assert.Equal(t, "0x80244022", formatWULastError(0x80244022))
	assert.Equal(t, "0xD", formatWULastError(13))
}
