// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

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
	assert.Equal(t, "[::]", ports[0].LocalAddress)
	assert.Equal(t, int64(0), ports[0].RemotePort)
	assert.Equal(t, "[::]", ports[0].RemoteAddress)
}

// TestParseWindowsTCPSlim covers the projected output shape produced by
//
//	@(Get-NetTCPConnection | Select-Object LocalAddress, LocalPort,
//	  RemoteAddress, RemotePort, @{Name='State';Expression={[int]$_.State}},
//	  OwningProcess) | ConvertTo-Json
//
// i.e. only the six fields we read, State as an int, always an array. This is
// what listWindows now asks PowerShell for to keep the payload small.
func TestParseWindowsTCPSlim(t *testing.T) {
	data, err := os.Open("./testdata/windows_tcp_slim.json")
	require.NoError(t, err)
	defer data.Close()

	ports, err := ParseWindowsNetTCPConnections(data)
	require.NoError(t, err)
	require.Equal(t, 2, len(ports))

	// ipv6 listener — the address gets bracketed
	assert.Equal(t, "[::]", ports[0].LocalAddress)
	assert.Equal(t, int64(49672), ports[0].LocalPort)
	assert.Equal(t, "[::]", ports[0].RemoteAddress)
	assert.Equal(t, int64(0), ports[0].RemotePort)
	assert.Equal(t, State(Listen), ports[0].State)
	assert.Equal(t, int64(2136), ports[0].OwningProcess)

	// ipv4 established connection — the address is left as-is
	assert.Equal(t, "10.0.0.5", ports[1].LocalAddress)
	assert.Equal(t, int64(52014), ports[1].LocalPort)
	assert.Equal(t, "93.184.216.34", ports[1].RemoteAddress)
	assert.Equal(t, int64(443), ports[1].RemotePort)
	assert.Equal(t, State(Established), ports[1].State)
	assert.Equal(t, int64(4288), ports[1].OwningProcess)
}
