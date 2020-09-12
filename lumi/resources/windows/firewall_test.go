package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWindowsFirewallSettings(t *testing.T) {
	data, err := os.Open("./testdata/firewall-settings.json")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := ParseWindowsFirewallSettings(data)
	assert.Nil(t, err)
	assert.Equal(t, int64(65535), settings.ActiveProfile)
}

func TestWindowsFirewallProfiles(t *testing.T) {
	data, err := os.Open("./testdata/firewall-profiles.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseWindowsFirewallProfiles(data)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}

func TestWindowsFirewallRules(t *testing.T) {
	data, err := os.Open("./testdata/firewall-rules.json")
	if err != nil {
		t.Fatal(err)
	}

	items, err := ParseWindowsFirewallRules(data)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}
