package procfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseProcessStatus(t *testing.T) {
	path := "./testdata/process-pid1.toml"
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: path})

	f, err := trans.FS().Open("/proc/1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	processStatus, err := ParseProcessStatus(f)
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}

	assert.NotNil(t, processStatus, "process is not nil")
	assert.Equal(t, "bash", processStatus.Executable, "detected process name")
}

func TestParseProcessCmdline(t *testing.T) {
	path := "./testdata/process-pid1.toml"
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: path})

	f, err := trans.FS().Open("/proc/1/cmdline")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cmd, err := ParseProcessCmdline(f)
	if err != nil {
		t.Fatalf("cannot request file %v", err)
	}
	assert.Equal(t, "/bin/bash", cmd, "detected process name")
}
