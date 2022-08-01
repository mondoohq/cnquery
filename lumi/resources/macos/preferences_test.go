package macos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestPreferences(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/user_preferences.toml")
	require.NoError(t, err)

	prefs := &Preferences{
		transport: mock,
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
