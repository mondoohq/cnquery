package macos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestPreferences(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_MOCK,
		Path:    "./testdata/osx.toml",
	})
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
