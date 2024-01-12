// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package macos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestPreferences(t *testing.T) {
	mock, err := mock.New("./testdata/user_preferences.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "macos",
			Version: "13.0",
			Family:  []string{"macos"},
		},
	})
	require.NoError(t, err)

	prefs := &Preferences{
		connection: mock,
	}

	preferences, err := prefs.UserHostPreferences()
	require.NoError(t, err)
	assert.NotNil(t, preferences["com.apple.Bluetooth"])
	assert.NotNil(t, preferences["com.apple.MIDI"])

	preferences, err = prefs.UserPreferences()
	require.NoError(t, err)
	assert.NotNil(t, preferences["com.apple.iCal.helper"])
	assert.NotNil(t, preferences["com.apple.iChat"])
}
