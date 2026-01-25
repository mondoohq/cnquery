// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package services

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// GetNativeWindowsServices uses the Windows Service Control Manager API directly
// to enumerate all services. This is significantly faster than PowerShell (~1-10ms vs 200-500ms).
func GetNativeWindowsServices() ([]*Service, error) {
	log.Debug().Msg("listing services using native Windows SCM API")

	// Open the Service Control Manager with enumerate access
	scm, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer scm.Disconnect()

	// List all services (both Win32 and driver services)
	serviceNames, err := scm.ListServices()
	if err != nil {
		return nil, err
	}

	services := make([]*Service, 0, len(serviceNames))

	for _, name := range serviceNames {
		svc, err := getServiceInfo(scm, name)
		if err != nil {
			// Log but continue - some services may be inaccessible
			log.Debug().Str("service", name).Err(err).Msg("could not get service info")
			continue
		}
		services = append(services, svc)
	}

	log.Debug().Int("count", len(services)).Msg("native Windows services enumeration complete")
	return services, nil
}

func getServiceInfo(scm *mgr.Mgr, name string) (*Service, error) {
	// Open the service with query access
	s, err := scm.OpenService(name)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// Get service configuration (includes display name and start type)
	config, err := s.Config()
	if err != nil {
		return nil, err
	}

	// Get current service status
	status, err := s.Query()
	if err != nil {
		return nil, err
	}

	return &Service{
		Name:        name,
		Description: config.DisplayName,
		Installed:   true,
		Running:     isRunning(status.State),
		Enabled:     isEnabled(config.StartType),
		State:       mapState(status.State),
		Type:        "windows",
	}, nil
}

// isRunning checks if the service is currently running
func isRunning(state svc.State) bool {
	return state == svc.Running
}

// isEnabled checks if the service is enabled (will start at boot or on demand)
// Start types 0-3 are considered enabled (Boot, System, Automatic, Manual)
// Start type 4 is Disabled
func isEnabled(startType uint32) bool {
	return startType <= 3
}

// mapState converts Windows service state to our canonical State type
// Windows service states from https://docs.microsoft.com/en-us/windows/win32/api/winsvc/ns-winsvc-service_status
func mapState(state svc.State) State {
	switch state {
	case svc.Stopped:
		return ServiceStopped
	case svc.StartPending:
		return ServiceStartPending
	case svc.StopPending:
		return ServiceStopPending
	case svc.Running:
		return ServiceRunning
	case svc.ContinuePending:
		return ServiceContinuePending
	case svc.PausePending:
		return ServicePausePending
	case svc.Paused:
		return ServicePaused
	default:
		return ServiceUnknown
	}
}
