// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build unix

package providers

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatExitStatus_ExitCode(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 2")
	_ = cmd.Run()
	require.NotNil(t, cmd.ProcessState)
	assert.Equal(t, "code:2", formatExitStatus(cmd.ProcessState))
}

func TestFormatExitStatus_CleanExit(t *testing.T) {
	cmd := exec.Command("true")
	require.NoError(t, cmd.Run())
	assert.Equal(t, "code:0", formatExitStatus(cmd.ProcessState))
}

func TestFormatExitStatus_Signal(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())
	require.NoError(t, cmd.Process.Kill()) // SIGKILL, same as the OOM killer
	_ = cmd.Wait()
	require.NotNil(t, cmd.ProcessState)
	assert.Equal(t, "signal:SIGKILL", formatExitStatus(cmd.ProcessState))
}

func TestFormatExitStatus_Nil(t *testing.T) {
	assert.Equal(t, "unknown", formatExitStatus(nil))
}

func TestMaxRSSBytes(t *testing.T) {
	cmd := exec.Command("true")
	require.NoError(t, cmd.Run())
	assert.Greater(t, maxRSSBytes(cmd.ProcessState), int64(0))
	assert.Zero(t, maxRSSBytes(nil))
}

func TestProcessTracker_NoSubprocess(t *testing.T) {
	// Untracked (builtin providers, tests): no state, no panic.
	tracker := &processTracker{}
	assert.Nil(t, tracker.exitState())

	var p *RunningProvider = &RunningProvider{}
	assert.Nil(t, p.exitState())
}
