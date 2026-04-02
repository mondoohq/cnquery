// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/mountedfs"
)

func TestSystemdFSTimerManagerList(t *testing.T) {
	mgr := &SystemdFSTimerManager{
		Fs: mountedfs.NewMountedFs("testdata/systemd"),
	}

	timers, err := mgr.List()
	require.NoError(t, err)

	timerMap := map[string]*SystemdTimer{}
	for _, tm := range timers {
		timerMap[tm.Name] = tm
	}

	// logrotate.timer has [Install] section and is in timers.target.wants
	require.Contains(t, timerMap, "logrotate")
	assert.Equal(t, "Daily rotation of log files", timerMap["logrotate"].Description)
	assert.True(t, timerMap["logrotate"].Installed)
	assert.True(t, timerMap["logrotate"].Enabled)
	assert.False(t, timerMap["logrotate"].Static)
	assert.False(t, timerMap["logrotate"].Masked)
	assert.False(t, timerMap["logrotate"].Running) // FS can't know running state

	// fstrim.timer has [Install] but is NOT in any .wants directory
	require.Contains(t, timerMap, "fstrim")
	assert.Equal(t, "Discard unused blocks once a week", timerMap["fstrim"].Description)
	assert.True(t, timerMap["fstrim"].Installed)
	assert.False(t, timerMap["fstrim"].Enabled)
	assert.False(t, timerMap["fstrim"].Static)

	// static-check.timer has no [Install] section
	require.Contains(t, timerMap, "static-check")
	assert.True(t, timerMap["static-check"].Installed)
	assert.False(t, timerMap["static-check"].Enabled)
	assert.True(t, timerMap["static-check"].Static)
}

func TestSystemdFSTimerManagerGet(t *testing.T) {
	mgr := &SystemdFSTimerManager{
		Fs: mountedfs.NewMountedFs("testdata/systemd"),
	}

	timer, err := mgr.Get("logrotate")
	require.NoError(t, err)
	assert.Equal(t, "logrotate", timer.Name)
	assert.Equal(t, "Daily rotation of log files", timer.Description)
	assert.True(t, timer.Enabled)
}

func TestSystemdFSTimerManagerGetNotFound(t *testing.T) {
	mgr := &SystemdFSTimerManager{
		Fs: mountedfs.NewMountedFs("testdata/systemd"),
	}

	timer, err := mgr.Get("nonexistent")
	require.Nil(t, timer)
	require.ErrorIs(t, err, ErrServiceNotFound)
}

func TestSystemdFSTimerManagerShowProperties(t *testing.T) {
	mgr := &SystemdFSTimerManager{
		Fs: mountedfs.NewMountedFs("testdata/systemd"),
	}

	t.Run("timer with explicit Unit", func(t *testing.T) {
		props, err := mgr.ShowTimerProperties("fstrim")
		require.NoError(t, err)
		assert.Equal(t, "weekly", props["OnCalendar"])
		assert.Equal(t, "yes", props["Persistent"])
		assert.Equal(t, "fstrim.service", props["Unit"])
	})

	t.Run("timer with implicit Unit", func(t *testing.T) {
		props, err := mgr.ShowTimerProperties("logrotate")
		require.NoError(t, err)
		assert.Equal(t, "daily", props["OnCalendar"])
		assert.Equal(t, "yes", props["Persistent"])
		// No explicit Unit= in the file, so it defaults to <name>.service
		assert.Equal(t, "logrotate.service", props["Unit"])
	})

	t.Run("not found", func(t *testing.T) {
		_, err := mgr.ShowTimerProperties("nonexistent")
		require.ErrorIs(t, err, ErrServiceNotFound)
	})
}
