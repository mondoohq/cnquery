// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package processes_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/resources/processes"
)

func TestLinuxPSProcessParser(t *testing.T) {
	mock, err := mock.New("./testdata/debian.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"linux"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := processes.ParseLinuxPsResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(m), "detected the right amount of processes")

	assert.Equal(t, "/bin/bash", m[0].Command, "process command detected")
	assert.Equal(t, int64(1), m[0].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[0].Uid, "process uid detected")

	assert.Equal(t, "ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command", m[1].Command, "process command detected")
	assert.Equal(t, int64(46), m[1].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[1].Uid, "process uid detected")

	assert.Equal(t, "", m[2].Command, "process command matched against empty COMMAND column")
	assert.Equal(t, int64(3987), m[2].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[2].Uid, "process uid detected")
}

func TestOSxPSProcessParser(t *testing.T) {
	mock, err := mock.New("./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("ps Axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := processes.ParseLinuxPsResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 41, len(m), "detected the right amount of processes")

	assert.Equal(t, "/sbin/launchd", m[0].Command, "process command detected")
	assert.Equal(t, int64(1), m[0].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[0].Uid, "process uid detected")

	assert.Equal(t, "/usr/sbin/syslogd", m[1].Command, "process command detected")
	assert.Equal(t, int64(125), m[1].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[1].Uid, "process uid detected")
}

func TestUnixPSProcessParser(t *testing.T) {
	mock, err := mock.New("./testdata/freebsd12.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Family: []string{"unix"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,time,uid,command")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := processes.ParseUnixPsResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 41, len(m), "detected the right amount of processes")

	assert.Equal(t, "[kernel]", m[0].Command, "process command detected")
	assert.Equal(t, int64(0), m[0].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[0].Uid, "process uid detected")

	assert.Equal(t, "[Timer]", m[20].Command, "process command detected")
	assert.Equal(t, int64(88), m[20].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[20].Uid, "process uid detected")
}

func TestAixPSProcessParser(t *testing.T) {
	mock, err := mock.New("./testdata/aix72.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "aix",
			Family: []string{"unix"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("ps -A -o pid,pcpu,pmem,vsz,tty,time,uid,args")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := processes.ParseAixPsResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 27, len(m), "detected the right amount of processes")

	// search ssh
	var found *processes.ProcessEntry
	for i := range m {
		if strings.HasPrefix(m[i].Command, "sshd") {
			found = m[i]
		}
	}

	assert.Equal(t, "sshd: cecuser [priv]", found.Command, "process command detected")
	assert.Equal(t, int64(3670308), found.Pid, "process pid detected")
	assert.Equal(t, int64(0), found.Uid, "process uid detected")
}

func TestParseLinuxFind(t *testing.T) {
	fi, err := os.Open("./testdata/find_nginx_container.txt")
	require.NoError(t, err)
	defer fi.Close()

	scanner := bufio.NewScanner(fi)
	scanner.Scan()
	line := scanner.Text()
	pid, inode, err := processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(0), pid)
	require.Equal(t, int64(0), inode)

	scanner.Scan()
	line = scanner.Text()
	pid, inode, err = processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(0), pid)
	require.Equal(t, int64(0), inode)

	scanner.Scan()
	line = scanner.Text()
	pid, inode, err = processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(1), pid)
	require.Equal(t, int64(41866685), inode)

	scanner.Scan()
	line = scanner.Text()
	pid, inode, err = processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(0), pid)
	require.Equal(t, int64(0), inode)

	scanner.Scan()
	line = scanner.Text()
	pid, inode, err = processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(0), pid)
	require.Equal(t, int64(0), inode)

	scanner.Scan()
	line = scanner.Text()
	pid, inode, err = processes.ParseLinuxFindLine(line)
	require.NoError(t, err)
	require.Equal(t, int64(1), pid)
	require.Equal(t, int64(18472), inode)
}
