// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
)

func TestParseSystemdTimerUnitFiles(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"UNIT FILE                    STATE    PRESET",
		"apt-daily.timer              enabled  enabled",
		"apt-daily-upgrade.timer      enabled  enabled",
		"e2scrub_all.timer            disabled enabled",
		"fstrim.timer                 disabled enabled",
		"logrotate.timer              static   enabled",
		"man-db.timer                 masked   enabled",
		"",
		"6 unit files listed.",
		"",
	}, "\n"))

	timers, err := ParseSystemdTimerUnitFiles(input)
	require.NoError(t, err)
	require.Len(t, timers, 6)

	// enabled timer
	assert.Equal(t, "apt-daily", timers[0].Name)
	assert.True(t, timers[0].Installed)
	assert.True(t, timers[0].Enabled)
	assert.False(t, timers[0].Masked)
	assert.False(t, timers[0].Static)

	// disabled timer
	assert.Equal(t, "e2scrub_all", timers[2].Name)
	assert.False(t, timers[2].Enabled)

	// static timer
	assert.Equal(t, "logrotate", timers[4].Name)
	assert.True(t, timers[4].Static)
	assert.False(t, timers[4].Enabled)

	// masked timer
	assert.Equal(t, "man-db", timers[5].Name)
	assert.True(t, timers[5].Masked)
	assert.False(t, timers[5].Enabled)
}

func TestParseSystemdTimerListUnits(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"  UNIT                       LOAD      ACTIVE   SUB     DESCRIPTION",
		"  apt-daily.timer            loaded    active   waiting Daily apt download activities",
		"  apt-daily-upgrade.timer    loaded    active   waiting Daily apt upgrade and clean activities",
		"  fstrim.timer               loaded    inactive dead    Discard unused blocks once a week",
		"● missing.timer              not-found inactive dead    missing.timer",
		"",
		"LOAD   = ...",
		"ACTIVE = ...",
		"SUB    = ...",
		"",
		"4 loaded units listed.",
		"",
	}, "\n"))

	timers, err := ParseSystemdTimerListUnits(input)
	require.NoError(t, err)
	require.Len(t, timers, 4)

	assert.Equal(t, "apt-daily", timers["apt-daily"].Name)
	assert.True(t, timers["apt-daily"].Running)
	assert.True(t, timers["apt-daily"].Installed)
	assert.Equal(t, "Daily apt download activities", timers["apt-daily"].Description)

	assert.False(t, timers["fstrim"].Running)
	assert.True(t, timers["fstrim"].Installed)

	assert.False(t, timers["missing"].Running)
	assert.False(t, timers["missing"].Installed)
}

func TestSystemdTimerManagerList(t *testing.T) {
	const listFilesCmd = "systemctl list-unit-files --type timer --all"
	const listUnitsCmd = "systemctl list-units --type timer --all"

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"ubuntu", "linux"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			listFilesCmd: {
				Stdout: strings.Join([]string{
					"UNIT FILE                STATE    PRESET",
					"apt-daily.timer          enabled  enabled",
					"fstrim.timer             disabled enabled",
					"",
					"2 unit files listed.",
					"",
				}, "\n"),
			},
			listUnitsCmd: {
				Stdout: strings.Join([]string{
					"  UNIT              LOAD   ACTIVE   SUB     DESCRIPTION",
					"  apt-daily.timer   loaded active   waiting Daily apt download activities",
					"",
					"LOAD   = ...",
					"",
					"1 loaded units listed.",
					"",
				}, "\n"),
			},
		},
	}))
	require.NoError(t, err)

	mgr := &SystemdTimerManager{conn: mockConn}
	timers, err := mgr.List()
	require.NoError(t, err)
	require.Len(t, timers, 2)

	timerMap := map[string]*SystemdTimer{}
	for _, timer := range timers {
		timerMap[timer.Name] = timer
	}

	// apt-daily: in both lists, active
	assert.True(t, timerMap["apt-daily"].Enabled)
	assert.True(t, timerMap["apt-daily"].Running)
	assert.Equal(t, "Daily apt download activities", timerMap["apt-daily"].Description)

	// fstrim: only in unit-files, not running
	assert.False(t, timerMap["fstrim"].Enabled)
	assert.False(t, timerMap["fstrim"].Running)
}

func TestSystemdTimerManagerGet(t *testing.T) {
	const showCmd = "systemctl show --property=Id,LoadState,ActiveState,UnitFileState,Description apt-daily.timer"

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"ubuntu", "linux"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			showCmd: {
				Stdout: strings.Join([]string{
					"Id=apt-daily.timer",
					"Description=Daily apt download activities",
					"LoadState=loaded",
					"ActiveState=active",
					"UnitFileState=enabled",
					"",
				}, "\n"),
			},
		},
	}))
	require.NoError(t, err)

	mgr := &SystemdTimerManager{conn: mockConn}
	timer, err := mgr.Get("apt-daily")
	require.NoError(t, err)

	assert.Equal(t, "apt-daily", timer.Name)
	assert.Equal(t, "Daily apt download activities", timer.Description)
	assert.True(t, timer.Installed)
	assert.True(t, timer.Running)
	assert.True(t, timer.Enabled)
	assert.False(t, timer.Masked)
	assert.False(t, timer.Static)
}

func TestSystemdTimerManagerGetNotFound(t *testing.T) {
	const showCmd = "systemctl show --property=Id,LoadState,ActiveState,UnitFileState,Description missing.timer"

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"ubuntu", "linux"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			showCmd: {
				Stdout: strings.Join([]string{
					"Id=missing.timer",
					"Description=missing.timer",
					"LoadState=not-found",
					"ActiveState=inactive",
					"UnitFileState=",
					"",
				}, "\n"),
			},
		},
	}))
	require.NoError(t, err)

	mgr := &SystemdTimerManager{conn: mockConn}
	timer, err := mgr.Get("missing")
	require.Nil(t, timer)
	require.ErrorIs(t, err, ErrServiceNotFound)
}

func TestShowTimerProperties(t *testing.T) {
	const showCmd = "systemctl show --property=Unit,OnCalendar,Persistent apt-daily.timer"

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"ubuntu", "linux"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			showCmd: {
				Stdout: strings.Join([]string{
					"Unit=apt-daily.service",
					"OnCalendar=*-*-* 6,18:00",
					"Persistent=yes",
					"",
				}, "\n"),
			},
		},
	}))
	require.NoError(t, err)

	mgr := &SystemdTimerManager{conn: mockConn}
	props, err := mgr.ShowTimerProperties("apt-daily")
	require.NoError(t, err)

	assert.Equal(t, "apt-daily.service", props["Unit"])
	assert.Equal(t, "*-*-* 6,18:00", props["OnCalendar"])
	assert.Equal(t, "yes", props["Persistent"])
}
