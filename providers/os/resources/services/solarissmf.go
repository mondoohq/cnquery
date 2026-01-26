// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"bufio"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

// SolarisSmfServiceManager handles Solaris Service Management Facility (SMF)
// https://docs.oracle.com/cd/E23824_01/html/821-1451/hbrunlevels-25516.html
type SolarisSmfServiceManager struct {
	conn shared.Connection
}

func (s *SolarisSmfServiceManager) Name() string {
	return "Solaris Service Management Facility"
}

func (s *SolarisSmfServiceManager) List() ([]*Service, error) {
	cmd, err := s.conn.RunCommand("svcs -a")
	if err != nil {
		return nil, err
	}

	return ParseSolarisSmfServices(cmd.Stdout), nil
}

// smfService represents a parsed SMF service entry
type smfService struct {
	State string
	FMRI  string
}

// ParseSolarisSmfServices parses the output of `svcs -a`
// Example output:
//
//	STATE          STIME           FMRI
//	online         22:01:55        svc:/network/ssh:default
//	disabled       22:01:40        svc:/network/dns/client:default
//	legacy_run     22:02:03        lrc:/etc/rc2_d/S89PRESERVE
func ParseSolarisSmfServices(r io.Reader) []*Service {
	var services []*Service
	scanner := bufio.NewScanner(r)

	// Skip header line
	if scanner.Scan() {
		// First line is header: "STATE          STIME           FMRI"
	}

	for scanner.Scan() {
		line := scanner.Text()
		entry := parseSmfLine(line)
		if entry == nil {
			continue
		}

		running, enabled := smfStateToRunningEnabled(entry.State)
		services = append(services, &Service{
			Name:      entry.FMRI,
			Installed: true,
			Running:   running,
			Enabled:   enabled,
			State:     smfStateToServiceState(entry.State),
			Type:      "smf",
		})
	}

	return services
}

// parseSmfLine parses a single line from svcs -a output
// Format: STATE          STIME           FMRI
// Fields are whitespace-separated, with STATE and STIME being fixed-width
func parseSmfLine(line string) *smfService {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil
	}

	return &smfService{
		State: fields[0],
		FMRI:  fields[2],
	}
}

// normalizeSmfState strips the asterisk suffix from transitioning states
// (e.g., "online*" -> "online" when a service is starting/stopping)
func normalizeSmfState(state string) string {
	return strings.TrimSuffix(state, "*")
}

// smfStateToRunningEnabled maps SMF states to running and enabled booleans
// SMF states: online, offline, disabled, maintenance, degraded, legacy_run, incomplete, uninitialized
func smfStateToRunningEnabled(state string) (running, enabled bool) {
	switch normalizeSmfState(state) {
	case "online":
		return true, true
	case "degraded":
		// Running but in a degraded state
		return true, true
	case "legacy_run":
		// Legacy init.d script is running
		return true, true
	case "offline":
		// Enabled but not running (waiting for dependencies)
		return false, true
	case "maintenance":
		// Enabled but in error state
		return false, true
	case "incomplete":
		// Service is in progress or waiting
		return false, true
	case "disabled", "uninitialized":
		return false, false
	default:
		// Unknown state, assume not running
		return false, false
	}
}

// smfStateToServiceState maps SMF state to the generic ServiceState
func smfStateToServiceState(state string) State {
	switch normalizeSmfState(state) {
	case "online", "degraded", "legacy_run":
		return ServiceRunning
	case "offline", "disabled", "maintenance", "incomplete", "uninitialized":
		return ServiceStopped
	default:
		return ServiceUnknown
	}
}
