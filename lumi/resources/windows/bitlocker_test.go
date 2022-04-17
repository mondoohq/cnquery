package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitlockerStatusPowershell(t *testing.T) {
	r, err := os.Open("./testdata/bitlocker_status.json")
	require.NoError(t, err)

	bitlock, err := ParseWindowsBitlockerStatus(r)
	require.NoError(t, err)
	assert.True(t, len(bitlock) == 2)

	assert.Equal(t, "\\\\?\\Volume{1b7897f7-3916-496c-91de-704fde33dde9}\\", bitlock[0].DeviceID)
	assert.Equal(t, "C:", bitlock[0].DriveLetter)
	assert.Equal(t, "XTS_AES_128", bitlock[0].EncryptionMethod.Text)

	assert.Equal(t, "\\\\?\\Volume{0e4c91e2-80c2-4433-bf7f-31fb65330364}\\", bitlock[1].DeviceID)
	assert.Equal(t, "E:", bitlock[1].DriveLetter)
	assert.Equal(t, "NONE", bitlock[1].EncryptionMethod.Text)
}
