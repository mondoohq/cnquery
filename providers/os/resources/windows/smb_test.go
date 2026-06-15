// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWindowsSmbShares(t *testing.T) {
	// Array result.
	shares, err := ParseWindowsSmbShares(strings.NewReader(`[
		{"Name":"C$","Path":"C:\\","Description":"Default share","ScopeName":"*","ShareType":"FileSystemDirectory"},
		{"Name":"SAP_Export","Path":"D:\\exports\\sap","Description":"","ScopeName":"*","ShareType":"FileSystemDirectory"}
	]`))
	require.NoError(t, err)
	require.Len(t, shares, 2)
	require.Equal(t, WindowsSmbShare{Name: "C$", Path: "C:\\", Description: "Default share", ScopeName: "*", ShareType: "FileSystemDirectory"}, shares[0])
	require.Equal(t, "SAP_Export", shares[1].Name)

	// PowerShell emits a bare object for a single result.
	single, err := ParseWindowsSmbShares(strings.NewReader(`{"Name":"C$","Path":"C:\\","Description":"d","ScopeName":"*","ShareType":"FileSystemDirectory"}`))
	require.NoError(t, err)
	require.Len(t, single, 1)
	require.Equal(t, "C$", single[0].Name)

	// Empty output.
	empty, err := ParseWindowsSmbShares(strings.NewReader(""))
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestParseWindowsSmbSessions(t *testing.T) {
	sessions, err := ParseWindowsSmbSessions(strings.NewReader(`[
		{"ClientComputerName":"192.168.1.50","ClientUserName":"CORP\\alice","Dialect":"3.1.1","NumOpens":3}
	]`))
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, WindowsSmbSession{ClientComputerName: "192.168.1.50", ClientUserName: "CORP\\alice", Dialect: "3.1.1", NumOpens: 3}, sessions[0])
}

func TestParseWindowsSmbConnections(t *testing.T) {
	connections, err := ParseWindowsSmbConnections(strings.NewReader(`[
		{"ServerName":"fs01","ShareName":"SAP_Export","UserName":"CORP\\bob","Dialect":"3.1.1"}
	]`))
	require.NoError(t, err)
	require.Len(t, connections, 1)
	require.Equal(t, WindowsSmbConnection{ServerName: "fs01", ShareName: "SAP_Export", UserName: "CORP\\bob", Dialect: "3.1.1"}, connections[0])
}
