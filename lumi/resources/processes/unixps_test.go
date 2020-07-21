package processes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/processes"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestLinuxPSProcessParser(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/debian.toml"})
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
	assert.Equal(t, 2, len(m), "detected the right amount of processes")

	assert.Equal(t, "/bin/bash", m[0].Command, "process command detected")
	assert.Equal(t, int64(1), m[0].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[0].Uid, "process uid detected")

	assert.Equal(t, "ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command", m[1].Command, "process command detected")
	assert.Equal(t, int64(46), m[1].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[1].Uid, "process uid detected")
}

func TestOSxPSProcessParser(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/osx.toml"})
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
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/freebsd12.toml"})
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
