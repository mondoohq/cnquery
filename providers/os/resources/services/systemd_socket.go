// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// SystemdSocket represents a systemd socket unit.
type SystemdSocket struct {
	Name        string
	Description string
	Installed   bool
	Enabled     bool
	Masked      bool
	Static      bool
	Running     bool
}

// SystemdSocketManager queries systemd for socket units via systemctl commands.
type SystemdSocketManager struct {
	conn shared.Connection
}

func NewSystemdSocketManager(conn shared.Connection) *SystemdSocketManager {
	return &SystemdSocketManager{conn: conn}
}

func (m *SystemdSocketManager) List() ([]*SystemdSocket, error) {
	// Step 1: Get all socket unit files (provides Enabled/Masked/Static/Installed)
	cmdList, err := m.conn.RunCommand("systemctl list-unit-files --type socket --all")
	if err != nil {
		return nil, err
	}

	sockets, err := ParseSystemdSocketUnitFiles(cmdList.Stdout)
	if err != nil {
		return nil, err
	}

	// Step 2: Get running state from list-units (provides Running/Description)
	cmdUnits, err := m.conn.RunCommand("systemctl list-units --type socket --all")
	if err != nil {
		return nil, err
	}

	unitStates, err := ParseSystemdSocketListUnits(cmdUnits.Stdout)
	if err != nil {
		return nil, err
	}

	// Step 3: Merge
	for _, socket := range sockets {
		unitState, ok := unitStates[socket.Name]
		if !ok {
			continue
		}
		socket.Description = unitState.Description
		socket.Running = unitState.Running
		if !unitState.Installed {
			socket.Installed = false
		}
	}

	return sockets, nil
}

func (m *SystemdSocketManager) Get(name string) (*SystemdSocket, error) {
	name = ensureSystemdSocketUnit(name)
	cmd, err := m.conn.RunCommand(buildSystemdShowCommand([]string{name}))
	if err != nil {
		return nil, err
	}

	services, err := ParseServiceSystemDShow(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	normalized := normalizeSystemdSocketName(name)
	svc, ok := services[normalized]
	if !ok || !svc.Installed {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, normalized)
	}

	return &SystemdSocket{
		Name:        normalized,
		Description: svc.Description,
		Installed:   svc.Installed,
		Enabled:     svc.Enabled,
		Masked:      svc.Masked,
		Static:      svc.Static,
		Running:     svc.Running,
	}, nil
}

// ShowSocketProperties runs systemctl show for socket-specific properties.
func (m *SystemdSocketManager) ShowSocketProperties(name string) (map[string]string, error) {
	unit := ensureSystemdSocketUnit(name)
	cmd, err := m.conn.RunCommand(
		"systemctl show " + shared.ShellEscape(unit) + " --property=Triggers,Accept",
	)
	if err != nil {
		return nil, err
	}

	return parseShowProperties(cmd.Stdout)
}

// ShowSocketListenAddresses gets listen addresses from systemctl show Listen property.
// The Listen property has a complex format, so we parse it separately.
func (m *SystemdSocketManager) ShowSocketListenAddresses(name string) ([]string, error) {
	unit := ensureSystemdSocketUnit(name)
	cmd, err := m.conn.RunCommand(
		"systemctl show " + shared.ShellEscape(unit) + " --property=Listen",
	)
	if err != nil {
		return nil, err
	}

	return parseListenProperty(cmd.Stdout)
}

func ParseSystemdSocketUnitFiles(input io.Reader) ([]*SystemdSocket, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("error executing systemctl list-unit-files --type socket: %v", err)
	}

	var sockets []*SystemdSocket
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return sockets, nil
	}

	for _, line := range lines[1 : len(lines)-1] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.Contains(line, "unit files listed.") {
			continue
		}

		socket := &SystemdSocket{
			Name:      normalizeSystemdSocketName(fields[0]),
			Installed: true,
		}
		applySystemdSocketUnitFileState(socket, fields[1])
		sockets = append(sockets, socket)
	}

	return sockets, nil
}

func ParseSystemdSocketListUnits(input io.Reader) (map[string]*SystemdSocket, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("error reading systemctl list-units output: %v", err)
	}

	sockets := map[string]*SystemdSocket{}
	matches := SYSTEMD_LIST_UNITS_REGEX.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		unitName := match[1]
		if !strings.HasSuffix(unitName, ".socket") {
			continue
		}

		name := normalizeSystemdSocketName(unitName)
		sockets[name] = &SystemdSocket{
			Name:        name,
			Description: strings.TrimSpace(match[5]),
			Running:     match[3] == "active",
			Installed:   match[2] != "not-found",
		}
	}

	return sockets, nil
}

func normalizeSystemdSocketName(unit string) string {
	return strings.TrimSuffix(unit, ".socket")
}

func ensureSystemdSocketUnit(name string) string {
	if strings.HasSuffix(name, ".socket") {
		return name
	}
	return name + ".socket"
}

func applySystemdSocketUnitFileState(socket *SystemdSocket, unitFileState string) {
	socket.Enabled = unitFileState == "enabled" || unitFileState == "enabled-runtime"
	socket.Masked = strings.HasPrefix(unitFileState, "masked")
	socket.Static = unitFileState == "static"
}

// parseListenProperty parses the Listen= output from systemctl show.
// The format is: Listen=<path> (<type>)\nListen=<path> (<type>)...
// or on some systems: Listen={ path=<path> ; type=<type> }
func parseListenProperty(input io.Reader) ([]string, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "Listen=" {
			continue
		}

		// Strip "Listen=" prefix if present
		line = strings.TrimPrefix(line, "Listen=")
		if line == "" {
			continue
		}

		// Format: "/run/dbus/system_bus_socket (Stream)" or "{ path=/run/...; type=Stream }"
		if strings.HasPrefix(line, "{") {
			// Parse structured format: { path=<path> ; type=<type> }
			line = strings.Trim(line, "{ }")
			for _, part := range strings.Split(line, ";") {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "path=") {
					addresses = append(addresses, strings.TrimPrefix(part, "path="))
				}
			}
		} else {
			// Parse simple format: <address> (<type>)
			if idx := strings.LastIndex(line, " ("); idx > 0 {
				addresses = append(addresses, strings.TrimSpace(line[:idx]))
			} else if line != "" {
				addresses = append(addresses, line)
			}
		}
	}

	return addresses, nil
}
