package ports

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParseWindowsTCP(t *testing.T) {
	data, err := os.Open("./testdata/windows_tcp.json")
	require.NoError(t, err)

	ports, err := ParseWindowsNetTCPConnections(data)
	require.NoError(t, err)
	assert.Equal(t, 1, len(ports))

	assert.Equal(t, int64(49672), ports[0].LocalPort)
	assert.Equal(t, "::", ports[0].LocalAddress)
	assert.Equal(t, int64(0), ports[0].RemotePort)
	assert.Equal(t, "::", ports[0].RemoteAddress)
}
