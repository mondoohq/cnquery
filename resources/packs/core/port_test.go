package core

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParseLinuxProcNetIPv4(t *testing.T) {
	fi, err := os.Open("./ports/testdata/tcp4.txt")
	require.NoError(t, err)
	defer fi.Close()

	scanner := bufio.NewScanner(fi)
	scanner.Scan()
	line := scanner.Text()
	port, err := parseProcNetLine(line)
	require.NoError(t, err)
	require.Nil(t, port)

	scanner.Scan()
	line = scanner.Text()
	port, err = parseProcNetLine(line)
	require.NoError(t, err)
	require.NotNil(t, port)

	assert.Equal(t, int64(53), (*port).Port)
	assert.Equal(t, "127.0.0.53", port.Address)
	assert.Equal(t, int64(0), port.RemotePort)
	assert.Equal(t, "0.0.0.0", port.RemoteAddress)

	scanner.Scan()
	scanner.Scan()
	line = scanner.Text()
	port, err = parseProcNetLine(line)
	require.NoError(t, err)
	require.NotNil(t, port)

	assert.Equal(t, int64(37200), (*port).Port)
	assert.Equal(t, "10.0.2.15", port.Address)
	assert.Equal(t, int64(80), port.RemotePort)
	assert.Equal(t, "185.125.190.36", port.RemoteAddress)
}

func TestParseLinuxProcNetIPv6(t *testing.T) {
	fi, err := os.Open("./ports/testdata/tcp6.txt")
	require.NoError(t, err)
	defer fi.Close()

	scanner := bufio.NewScanner(fi)
	scanner.Scan()
	line := scanner.Text()
	port, err := parseProcNetLine(line)
	require.NoError(t, err)
	require.Nil(t, port)

	scanner.Scan()
	line = scanner.Text()
	port, err = parseProcNetLine(line)
	require.NoError(t, err)
	require.NotNil(t, port)

	assert.Equal(t, int64(22), (*port).Port)
	assert.Equal(t, "::", port.Address)
	assert.Equal(t, int64(0), port.RemotePort)
	assert.Equal(t, "::", port.RemoteAddress)
}
