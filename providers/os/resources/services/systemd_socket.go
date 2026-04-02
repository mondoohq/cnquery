// Copyright Mondoo, Inc. 2024, 2026
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
	unit := ensureSystemdSocketUnit(name)
	cmd, err := m.conn.RunCommand(buildSystemdShowCommand([]string{unit}))
	if err != nil {
		return nil, err
	}

	props, err := parseShowProperties(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	id := props["Id"]
	if id == "" || props["LoadState"] == "not-found" || props["LoadState"] == "" {
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, name)
	}

	socket := &SystemdSocket{
		Name:        normalizeSystemdSocketName(id),
		Description: props["Description"],
		Installed:   true,
		Running:     props["ActiveState"] == "active",
	}
	applyUnitFileState(&socket.Enabled, &socket.Masked, &socket.Static, props["UnitFileState"])

	return socket, nil
}

// ShowSocketProperties runs systemctl show for socket-specific properties
// including Listen for addresses, Triggers for the activated unit, and Accept.
func (m *SystemdSocketManager) ShowSocketProperties(name string) (map[string]string, error) {
	unit := ensureSystemdSocketUnit(name)
	cmd, err := m.conn.RunCommand(buildShowPropertyCommand("Triggers,Accept,Listen", unit))
	if err != nil {
		return nil, err
	}

	return parseShowProperties(cmd.Stdout)
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

	for _, line := range lines[1:] {
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
		applyUnitFileState(&socket.Enabled, &socket.Masked, &socket.Static, fields[1])
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

// ParseListenProperty parses the Listen= value from systemctl show output.
// Multi-listen sockets produce multiple Listen= lines which are joined with "\n"
// by parseShowProperties. Each line is one of:
//   - "/run/dbus/system_bus_socket (Stream)" — simple format
//   - "{ path=/run/...; type=Stream }" — structured format
func ParseListenProperty(listen string) []string {
	if listen == "" {
		return nil
	}

	var addresses []string
	for _, line := range strings.Split(listen, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Structured format: "{ path=<path> ; type=<type> }"
		if strings.HasPrefix(line, "{") {
			inner := strings.Trim(line, "{ }")
			for _, part := range strings.Split(inner, ";") {
				part = strings.TrimSpace(part)
				if after, ok := strings.CutPrefix(part, "path="); ok {
					addresses = append(addresses, after)
				}
			}
			continue
		}

		// Simple format: "<address> (<type>)"
		if idx := strings.LastIndex(line, " ("); idx > 0 {
			addresses = append(addresses, strings.TrimSpace(line[:idx]))
		} else {
			addresses = append(addresses, line)
		}
	}

	if len(addresses) == 0 {
		return nil
	}
	return addresses
}
