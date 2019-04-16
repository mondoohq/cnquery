package procfs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/procfs"
	"go.mondoo.io/mondoo/motor/mock/toml"
	"go.mondoo.io/mondoo/motor/types"
)

func TestParseProcessStatus(t *testing.T) {
	path := "./process-pid1.toml"
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: path})

	statusf, err := trans.File("/proc/1/status")
	if err != nil {
		t.Fatal(err)
	}

	statusStream, err := statusf.Open()
	if err != nil {
		t.Fatal(err)
	}

	processStatus, err := procfs.ParseProcessStatus(statusStream)
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, processStatus, "process is not nil")
	assert.Equal(t, "bash", processStatus.Executable, "detected process name")
}

func TestParseProcessCmdline(t *testing.T) {
	path := "./process-pid1.toml"
	trans, err := toml.New(&types.Endpoint{Backend: "mock", Path: path})

	f, err := trans.File("/proc/1/cmdline")
	if err != nil {
		t.Fatal(err)
	}

	cmdlineStream, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := procfs.ParseProcessCmdline(cmdlineStream)
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}
	assert.Equal(t, "/bin/bash", cmd, "detected process name")
}
