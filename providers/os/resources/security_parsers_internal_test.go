// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEtcTable(t *testing.T) {
	content := "# a comment\n127.0.0.1\tlocalhost loopback   # local\n\n" +
		"::1 localhost\n   \n10.0.0.5   host.example.com host\n"
	rows := parseEtcTable(content)
	require.Len(t, rows, 3)

	assert.Equal(t, 2, rows[0].line)
	assert.Equal(t, []string{"127.0.0.1", "localhost", "loopback"}, rows[0].fields)
	assert.Equal(t, "local", rows[0].comment)

	assert.Equal(t, 4, rows[1].line)
	assert.Equal(t, []string{"::1", "localhost"}, rows[1].fields)

	assert.Equal(t, 6, rows[2].line)
	assert.Equal(t, []string{"10.0.0.5", "host.example.com", "host"}, rows[2].fields)
}

func TestParsePlainHistory(t *testing.T) {
	// bash without HISTTIMEFORMAT: plain lines, no timestamps
	cmds := parsePlainHistory("ls -la\ncd /tmp\n")
	require.Len(t, cmds, 2)
	assert.Equal(t, "ls -la", cmds[0].command)
	assert.Nil(t, cmds[0].ts)

	// bash with HISTTIMEFORMAT: "#<epoch>" applies to the next command
	cmds = parsePlainHistory("#1700000000\nwhoami\nid\n")
	require.Len(t, cmds, 2)
	assert.Equal(t, "whoami", cmds[0].command)
	require.NotNil(t, cmds[0].ts)
	assert.Equal(t, int64(1700000000), cmds[0].ts.Unix())
	assert.Equal(t, "id", cmds[1].command)
	assert.Nil(t, cmds[1].ts) // marker only applies to the immediately following command
}

func TestParseZshHistory(t *testing.T) {
	// EXTENDED_HISTORY carries an epoch; a plain line does not
	cmds := parseZshHistory(": 1700000000:0;git status\nls\n")
	require.Len(t, cmds, 2)
	assert.Equal(t, "git status", cmds[0].command)
	require.NotNil(t, cmds[0].ts)
	assert.Equal(t, int64(1700000000), cmds[0].ts.Unix())
	assert.Equal(t, "ls", cmds[1].command)
	assert.Nil(t, cmds[1].ts)
}

func TestParseFishHistory(t *testing.T) {
	cmds := parseFishHistory("- cmd: echo hi\n  when: 1700000000\n- cmd: pwd\n")
	require.Len(t, cmds, 2)
	assert.Equal(t, "echo hi", cmds[0].command)
	require.NotNil(t, cmds[0].ts)
	assert.Equal(t, int64(1700000000), cmds[0].ts.Unix())
	assert.Equal(t, "pwd", cmds[1].command)
	assert.Nil(t, cmds[1].ts)
}

func TestParseDriverDate(t *testing.T) {
	// PS 5.1 / .NET "/Date(ms)/" with a timezone offset
	ts := parseDriverDate(json.RawMessage(`"/Date(1150848000000+0000)/"`))
	require.NotNil(t, ts)
	assert.Equal(t, int64(1150848000), ts.Unix())

	// PS 7 ISO-8601
	ts = parseDriverDate(json.RawMessage(`"2006-06-21T00:00:00Z"`))
	require.NotNil(t, ts)
	assert.Equal(t, int64(1150848000), ts.Unix())

	// null / object / empty -> no timestamp (never errors)
	assert.Nil(t, parseDriverDate(json.RawMessage(`null`)))
	assert.Nil(t, parseDriverDate(json.RawMessage(`{"foo":1}`)))
	assert.Nil(t, parseDriverDate(nil))
}
