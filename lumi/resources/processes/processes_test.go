package processes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/processes"
	mock "go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestPSProcessParser(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "processes_unix.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("ps axo pid,pcpu,pmem,vsz,rss,tty,stat,stime,time,uid,command")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := processes.ParseUnixPsResult(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 83, len(m), "detected the right amount of processes")

	assert.Equal(t, "/usr/lib/systemd/systemd --switched-root --system --deserialize 21", m[0].Command, "process command detected")
	assert.Equal(t, int64(1), m[0].Pid, "process pid detected")
	assert.Equal(t, int64(0), m[0].Uid, "process uid detected")

	assert.Equal(t, "/bin/dbus-daemon --system --address=systemd: --nofork --nopidfile --systemd-activation", m[65].Command, "process command detected")
	assert.Equal(t, int64(557), m[65].Pid, "process pid detected")
	assert.Equal(t, int64(81), m[65].Uid, "process uid detected")
}
