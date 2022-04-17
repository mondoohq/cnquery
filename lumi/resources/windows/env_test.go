package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseEnv(t *testing.T) {
	r, err := os.Open("./testdata/env.json")
	require.NoError(t, err)

	items, err := ParseEnv(r)
	assert.Nil(t, err)
	assert.Equal(t, 9, len(items))

	assert.Equal(t, "C:\\Windows\\system32;C:\\Windows;C:\\Windows\\System32\\Wbem;C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\;C:\\Windows\\System32\\OpenSSH\\;C:\\Program Files\\Amazon\\cfn-bootstrap\\;C:\\Windows\\system32\\config\\systemprofile\\AppData\\Local\\Microsoft\\WindowsApps;C:\\Users\\Administrator\\AppData\\Local\\Microsoft\\WindowsApps;", items["Path"])
}
