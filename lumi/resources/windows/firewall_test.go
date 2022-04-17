package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsFirewallSettings(t *testing.T) {
	r, err := os.Open("./testdata/firewall-settings.json")
	require.NoError(t, err)

	settings, err := ParseWindowsFirewallSettings(r)
	assert.Nil(t, err)
	assert.Equal(t, int64(65535), settings.ActiveProfile)
}

func TestWindowsFirewallProfiles(t *testing.T) {
	r, err := os.Open("./testdata/firewall-profiles.json")
	require.NoError(t, err)

	items, err := ParseWindowsFirewallProfiles(r)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}

func TestWindowsFirewallRules(t *testing.T) {
	r, err := os.Open("./testdata/firewall-rules.json")
	require.NoError(t, err)

	items, err := ParseWindowsFirewallRules(r)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}
