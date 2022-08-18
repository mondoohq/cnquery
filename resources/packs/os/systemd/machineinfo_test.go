package systemd

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestParseMachineInfo(t *testing.T) {

	content := `
PRETTY_HOSTNAME="Lennart's Tablet"
ICON_NAME=computer-tablet
CHASSIS=tablet
DEPLOYMENT=production
`

	mi, err := ParseMachineInfo(strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, "Lennart's Tablet", mi.PrettyHostname)
	assert.Equal(t, "computer-tablet", mi.IconName)
	assert.Equal(t, "tablet", mi.Chassis)
	assert.Equal(t, "production", mi.Deployment)
}
