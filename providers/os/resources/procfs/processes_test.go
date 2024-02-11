// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package procfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestParseProcessStatus(t *testing.T) {
	trans, err := mock.New(0, "./testdata/process-pid1.toml", nil)
	require.NoError(t, err)

	f, err := trans.FileSystem().Open("/proc/1/status")
	require.NoError(t, err)
	defer f.Close()

	processStatus, err := ParseProcessStatus(f)
	require.NoError(t, err)

	assert.NotNil(t, processStatus, "process is not nil")
	assert.Equal(t, "bash", processStatus.Executable, "detected process name")
}

func TestParseProcessCmdline(t *testing.T) {
	trans, err := mock.New(0, "./testdata/process-pid1.toml", nil)
	require.NoError(t, err)

	f, err := trans.FileSystem().Open("/proc/1/cmdline")
	require.NoError(t, err)
	defer f.Close()

	cmd, err := ParseProcessCmdline(f)
	require.NoError(t, err)
	assert.Equal(t, "/bin/bash", cmd, "detected process name")
}
