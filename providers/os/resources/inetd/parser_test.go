// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package inetd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	content := `# /etc/inetd.conf example
#
# <service> <socket> <proto> <wait> <user> <server> <args>
ftp     stream  tcp     nowait  root    /usr/sbin/tcpd  in.ftpd
telnet  stream  tcp     nowait  root    /usr/sbin/tcpd  in.telnetd
#shell  stream  tcp     nowait  root    /usr/sbin/tcpd  in.rshd
daytime stream  tcp     nowait  root    internal

   # indented comment is still a comment
tftp    dgram   udp     wait    nobody  /usr/sbin/tcpd  in.tftpd /srv/tftp
ftp     stream  tcp6    nowait  root    /usr/sbin/tcpd  in.ftpd
`

	entries := Parse(content)
	require.Len(t, entries, 5)

	t.Run("first active entry maps all columns", func(t *testing.T) {
		ftp := entries[0]
		assert.Equal(t, "ftp", ftp.Name)
		assert.Equal(t, "stream", ftp.SocketType)
		assert.Equal(t, "tcp", ftp.Protocol)
		assert.Equal(t, "nowait", ftp.Wait)
		assert.Equal(t, "root", ftp.User)
		assert.Equal(t, "/usr/sbin/tcpd", ftp.Server)
		assert.Equal(t, "in.ftpd", ftp.Arguments)
		assert.Equal(t, 4, ftp.Line)
	})

	t.Run("commented-out entries are excluded", func(t *testing.T) {
		for _, e := range entries {
			assert.NotEqual(t, "shell", e.Name)
		}
	})

	t.Run("internal servers have no arguments", func(t *testing.T) {
		daytime := entries[2]
		assert.Equal(t, "daytime", daytime.Name)
		assert.Equal(t, "internal", daytime.Server)
		assert.Equal(t, "", daytime.Arguments)
	})

	t.Run("multiple arguments are joined", func(t *testing.T) {
		tftp := entries[3]
		assert.Equal(t, "tftp", tftp.Name)
		assert.Equal(t, "in.tftpd /srv/tftp", tftp.Arguments)
	})

	t.Run("same service over different protocols both kept", func(t *testing.T) {
		assert.Equal(t, "ftp", entries[4].Name)
		assert.Equal(t, "tcp6", entries[4].Protocol)
	})
}

func TestParseEmpty(t *testing.T) {
	assert.Empty(t, Parse(""))
	assert.Empty(t, Parse("# only comments\n#ftp stream tcp nowait root internal\n"))
}

func TestParseRPCAndSuffixes(t *testing.T) {
	content := `rstatd/1-3      dgram   rpc/udp wait    root    /usr/sbin/rpc.rstatd    rpc.rstatd
echo            stream  tcp     nowait.400      root    internal
finger          stream  tcp     nowait  nobody.nogroup  /usr/sbin/tcpd  in.fingerd`

	entries := Parse(content)
	require.Len(t, entries, 3)

	assert.Equal(t, "rstatd/1-3", entries[0].Name)
	assert.Equal(t, "rpc/udp", entries[0].Protocol)

	// nowait.max suffix is preserved verbatim
	assert.Equal(t, "nowait.400", entries[1].Wait)

	// user.group suffix is preserved verbatim
	assert.Equal(t, "nobody.nogroup", entries[2].User)
}

func TestParseMalformedLinesSkipped(t *testing.T) {
	content := `ftp stream tcp nowait root
ftp stream tcp nowait root /usr/sbin/in.ftpd`

	entries := Parse(content)
	require.Len(t, entries, 1)
	assert.Equal(t, "/usr/sbin/in.ftpd", entries[0].Server)
	assert.Equal(t, 2, entries[0].Line)
}
