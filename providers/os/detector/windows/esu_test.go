// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsESUStatus(t *testing.T) {
	t.Run("subscription eligible", func(t *testing.T) {
		data := `{"SubscriptionEligible":true,"LicenseActivated":false}`

		status, err := ParseWindowsESUStatus(strings.NewReader(data))
		require.NoError(t, err)
		assert.True(t, status.SubscriptionEligible)
		assert.False(t, status.LicenseActivated)
		assert.True(t, status.ESUEnabled())
	})

	t.Run("MAK license activated", func(t *testing.T) {
		data := `{"SubscriptionEligible":false,"LicenseActivated":true}`

		status, err := ParseWindowsESUStatus(strings.NewReader(data))
		require.NoError(t, err)
		assert.False(t, status.SubscriptionEligible)
		assert.True(t, status.LicenseActivated)
		assert.True(t, status.ESUEnabled())
	})

	t.Run("both subscription and MAK", func(t *testing.T) {
		data := `{"SubscriptionEligible":true,"LicenseActivated":true}`

		status, err := ParseWindowsESUStatus(strings.NewReader(data))
		require.NoError(t, err)
		assert.True(t, status.SubscriptionEligible)
		assert.True(t, status.LicenseActivated)
		assert.True(t, status.ESUEnabled())
	})

	t.Run("ESU not enabled", func(t *testing.T) {
		data := `{"SubscriptionEligible":false,"LicenseActivated":false}`

		status, err := ParseWindowsESUStatus(strings.NewReader(data))
		require.NoError(t, err)
		assert.False(t, status.SubscriptionEligible)
		assert.False(t, status.LicenseActivated)
		assert.False(t, status.ESUEnabled())
	})

	t.Run("empty output", func(t *testing.T) {
		status, err := ParseWindowsESUStatus(strings.NewReader(""))
		require.NoError(t, err)
		assert.False(t, status.ESUEnabled())
	})
}
