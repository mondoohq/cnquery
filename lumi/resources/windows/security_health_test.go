package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSecurityHealthPowershell(t *testing.T) {
	r, err := os.Open("./testdata/security_center_health.json")
	require.NoError(t, err)

	health, err := ParseSecurityProviderHealth(r)
	require.NoError(t, err)

	assert.Equal(t, int64(2), health.Firewall.Code)
	assert.Equal(t, "POOR", health.Firewall.Text)
	assert.Equal(t, int64(0), health.AutoUpdate.Code)
	assert.Equal(t, "GOOD", health.AutoUpdate.Text)
	assert.Equal(t, int64(2), health.Uac.Code)
	assert.Equal(t, "POOR", health.Uac.Text)
}
