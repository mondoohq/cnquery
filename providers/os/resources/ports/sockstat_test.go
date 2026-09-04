// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package ports

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Captured verbatim from `sockstat -46 -s` on FreeBSD 14.4-RELEASE.
const sockstatFreeBSD14 = `USER     COMMAND    PID   FD  PROTO     LOCAL ADDRESS         FOREIGN ADDRESS       CONN STATE
ec2-user sshd-sessi 10356 7   tcp4      10.45.1.14:22         97.115.90.143:61073   ESTABLISHED
root     sshd        1718 6   tcp6      *:22                  *:*                   LISTEN
root     sshd        1718 7   tcp4      *:22                  *:*                   LISTEN
ntpd     ntpd        1666 21  udp4      *:123                 *:*
ntpd     ntpd        1666 22  udp4      10.45.1.14:123        *:*
ntpd     ntpd        1666 23  udp6      [::1]:123             *:*
ntpd     ntpd        1666 24  udp6      [fe80::1%lo0]:123     *:*
`

func TestParseSockstatListeners(t *testing.T) {
	entries, err := ParseSockstat(strings.NewReader(sockstatFreeBSD14))
	require.NoError(t, err)
	require.Len(t, entries, 7, "the header must not become a row")

	// tcp6 listener on the wildcard address
	e := entries[1]
	assert.Equal(t, "root", e.User)
	assert.Equal(t, "sshd", e.Command)
	assert.Equal(t, int64(1718), e.Pid)
	assert.Equal(t, "tcp6", e.Protocol)
	assert.Equal(t, "*", e.LocalAddress)
	assert.Equal(t, int64(22), e.LocalPort)
	assert.Equal(t, "", e.RemoteAddress, "*:* is not a peer")
	assert.Equal(t, int64(0), e.RemotePort)
	assert.Equal(t, "LISTEN", e.State)
}

func TestParseSockstatEstablished(t *testing.T) {
	entries, err := ParseSockstat(strings.NewReader(sockstatFreeBSD14))
	require.NoError(t, err)

	e := entries[0]
	assert.Equal(t, "tcp4", e.Protocol)
	assert.Equal(t, "10.45.1.14", e.LocalAddress)
	assert.Equal(t, int64(22), e.LocalPort)
	assert.Equal(t, "97.115.90.143", e.RemoteAddress)
	assert.Equal(t, int64(61073), e.RemotePort)
	assert.Equal(t, "ESTABLISHED", e.State)
}

// udp rows carry no CONN STATE column, and IPv6 literals bring their own
// colons plus an optional zone.
func TestParseSockstatUDPAndIPv6(t *testing.T) {
	entries, err := ParseSockstat(strings.NewReader(sockstatFreeBSD14))
	require.NoError(t, err)

	udp6 := entries[5]
	assert.Equal(t, "udp6", udp6.Protocol)
	assert.Equal(t, "::1", udp6.LocalAddress, "brackets are stripped")
	assert.Equal(t, int64(123), udp6.LocalPort)
	assert.Equal(t, "", udp6.State, "udp has no connection state")

	zoned := entries[6]
	assert.Equal(t, "fe80::1%lo0", zoned.LocalAddress, "the zone is part of the address")
	assert.Equal(t, int64(123), zoned.LocalPort)
}

// Older releases have no -s flag, so the state column is simply absent.
func TestParseSockstatWithoutStateColumn(t *testing.T) {
	raw := `USER     COMMAND    PID   FD  PROTO     LOCAL ADDRESS         FOREIGN ADDRESS
root     sshd        1718 7   tcp4      *:22                  *:*
`
	entries, err := ParseSockstat(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(22), entries[0].LocalPort)
	assert.Equal(t, "", entries[0].State)
}

// Unix domain sockets and other non-inet rows are not ports.
func TestParseSockstatSkipsNonInet(t *testing.T) {
	raw := `USER     COMMAND    PID   FD  PROTO     LOCAL ADDRESS         FOREIGN ADDRESS
root     dbus        900  3   stream    /var/run/dbus/system_bus_socket  -
root     sshd        1718 7   tcp4      *:22                  *:*
`
	entries, err := ParseSockstat(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "tcp4", entries[0].Protocol)
}

// A kernel socket has "?" where the pid goes; the row is still a real socket.
func TestParseSockstatKernelSocket(t *testing.T) {
	raw := `USER     COMMAND    PID   FD  PROTO     LOCAL ADDRESS         FOREIGN ADDRESS       CONN STATE
?        ?           ?    ?   tcp4      *:2049                *:*                   LISTEN
`
	entries, err := ParseSockstat(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(0), entries[0].Pid)
	assert.Equal(t, int64(2049), entries[0].LocalPort)
}

func TestParseSockstatEmpty(t *testing.T) {
	entries, err := ParseSockstat(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		in   string
		host string
		port int64
	}{
		{"*:22", "*", 22},
		{"10.0.0.1:443", "10.0.0.1", 443},
		{"[::1]:123", "::1", 123},
		{"[fe80::1%lo0]:123", "fe80::1%lo0", 123},
		{"*:*", "", 0},
		{"", "", 0},
		{"*:sunrpc", "*", 0},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			h, p := splitHostPort(tc.in)
			assert.Equal(t, tc.host, h)
			assert.Equal(t, tc.port, p)
		})
	}
}
