// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/svc"
)

func TestGetNativeWindowsServices(t *testing.T) {
	services, err := GetNativeWindowsServices()
	require.NoError(t, err)
	assert.NotEmpty(t, services, "expected at least one service")

	// Verify service structure is populated correctly
	for _, s := range services {
		assert.NotEmpty(t, s.Name, "service name should not be empty")
		assert.Equal(t, "windows", s.Type, "service type should be 'windows'")
		assert.True(t, s.Installed, "service should be marked as installed")
		// State should be one of the known states
		assert.Contains(t, []State{
			ServiceStopped,
			ServiceStartPending,
			ServiceStopPending,
			ServiceRunning,
			ServiceContinuePending,
			ServicePausePending,
			ServicePaused,
			ServiceUnknown,
		}, s.State, "service state should be a known state")
	}
}

func TestGetNativeWindowsServices_WellKnownServices(t *testing.T) {
	services, err := GetNativeWindowsServices()
	require.NoError(t, err)

	// These services should exist on all Windows systems
	wellKnownServices := []string{
		"Winmgmt",    // Windows Management Instrumentation
		"EventLog",   // Windows Event Log
		"PlugPlay",   // Plug and Play
		"RpcSs",      // Remote Procedure Call (RPC)
		"Schedule",   // Task Scheduler
		"SENS",       // System Event Notification Service
		"Spooler",    // Print Spooler (may be disabled but should exist)
		"W32Time",    // Windows Time
		"Dhcp",       // DHCP Client
		"Dnscache",   // DNS Client
		"LanmanServer", // Server
		"LanmanWorkstation", // Workstation
	}

	serviceMap := make(map[string]*Service)
	for _, s := range services {
		serviceMap[s.Name] = s
	}

	foundCount := 0
	for _, name := range wellKnownServices {
		if _, ok := serviceMap[name]; ok {
			foundCount++
		}
	}

	// At least half of the well-known services should be present
	// (some may be disabled/removed on minimal installations)
	assert.GreaterOrEqual(t, foundCount, len(wellKnownServices)/2,
		"expected at least %d well-known services, found %d", len(wellKnownServices)/2, foundCount)
}

func TestGetNativeWindowsServices_RunningServices(t *testing.T) {
	services, err := GetNativeWindowsServices()
	require.NoError(t, err)

	// At least some services should be running on any Windows system
	runningCount := 0
	for _, s := range services {
		if s.Running {
			runningCount++
			// Running services should have Running state
			assert.Equal(t, ServiceRunning, s.State,
				"service %s is marked running but state is %s", s.Name, s.State)
		}
	}

	assert.Greater(t, runningCount, 0, "expected at least one running service")
}

func TestMapState(t *testing.T) {
	tests := []struct {
		input    svc.State
		expected State
	}{
		{svc.Stopped, ServiceStopped},
		{svc.StartPending, ServiceStartPending},
		{svc.StopPending, ServiceStopPending},
		{svc.Running, ServiceRunning},
		{svc.ContinuePending, ServiceContinuePending},
		{svc.PausePending, ServicePausePending},
		{svc.Paused, ServicePaused},
		{svc.State(99), ServiceUnknown}, // Unknown state
	}

	for _, tc := range tests {
		t.Run(string(tc.expected), func(t *testing.T) {
			assert.Equal(t, tc.expected, mapState(tc.input))
		})
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name      string
		startType uint32
		expected  bool
	}{
		{"Boot", 0, true},
		{"System", 1, true},
		{"Automatic", 2, true},
		{"Manual", 3, true},
		{"Disabled", 4, false},
		{"Unknown high value", 5, false},
		{"Unknown higher value", 100, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isEnabled(tc.startType))
		})
	}
}

func TestIsRunning(t *testing.T) {
	tests := []struct {
		state    svc.State
		expected bool
	}{
		{svc.Stopped, false},
		{svc.StartPending, false},
		{svc.StopPending, false},
		{svc.Running, true},
		{svc.ContinuePending, false},
		{svc.PausePending, false},
		{svc.Paused, false},
	}

	for _, tc := range tests {
		t.Run(string(mapState(tc.state)), func(t *testing.T) {
			assert.Equal(t, tc.expected, isRunning(tc.state))
		})
	}
}
