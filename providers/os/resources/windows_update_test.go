// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
