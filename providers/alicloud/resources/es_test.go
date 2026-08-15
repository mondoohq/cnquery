// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	tea "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEsParseTime covers the cluster timestamp parser across the layouts the
// API uses. An unparseable value must stay null rather than becoming the zero
// time, which would report 1 January year 1 as a real creation date.
func TestEsParseTime(t *testing.T) {
	want := time.Date(2026, 8, 15, 12, 30, 0, 0, time.UTC)

	t.Run("nil stays nil", func(t *testing.T) {
		assert.Nil(t, esParseTime(nil))
	})
	t.Run("empty stays nil", func(t *testing.T) {
		assert.Nil(t, esParseTime(tea.String("")))
	})
	t.Run("garbage stays nil", func(t *testing.T) {
		assert.Nil(t, esParseTime(tea.String("not a date")))
	})
	t.Run("epoch milliseconds are not silently accepted", func(t *testing.T) {
		assert.Nil(t, esParseTime(tea.String("1562849679000")))
	})
	t.Run("rfc3339", func(t *testing.T) {
		got := esParseTime(tea.String("2026-08-15T12:30:00Z"))
		require.NotNil(t, got)
		assert.Equal(t, want, got.UTC())
	})
	t.Run("milliseconds layout", func(t *testing.T) {
		got := esParseTime(tea.String("2026-08-15T12:30:00.000Z"))
		require.NotNil(t, got)
		assert.Equal(t, want, got.UTC())
	})
}

// TestEsStrings covers the address-list flattening. A nil or empty entry must be
// dropped rather than surfacing as a blank string, which would read as a
// configured rule in a whitelist.
func TestEsStrings(t *testing.T) {
	assert.Equal(t, []any{}, esStrings(nil))
	assert.Equal(t, []any{}, esStrings([]*string{nil, tea.String("")}))
	assert.Equal(t, []any{"0.0.0.0/0"}, esStrings([]*string{tea.String("0.0.0.0/0"), nil}))
	assert.Equal(t, []any{"10.0.0.0/8", "192.168.0.0/16"},
		esStrings([]*string{tea.String("10.0.0.0/8"), tea.String(""), tea.String("192.168.0.0/16")}))
}

// TestEsInternetExposed covers the exposure verdict. Kibana counts on its own:
// the console reads everything the cluster holds, so treating a public console
// as unexposed because the cluster endpoint is private would miss the finding.
func TestEsInternetExposed(t *testing.T) {
	tests := []struct {
		name                 string
		public, kibanaPublic bool
		want                 bool
	}{
		{"both private", false, false, false},
		{"cluster endpoint public", true, false, true},
		{"only kibana public", false, true, true},
		{"both public", true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, esInternetExposed(tt.public, tt.kibanaPublic))
		})
	}
}
