// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// SystemdTimer represents a systemd timer unit.
type SystemdTimer struct {
	Name        string
	Description string
	Installed   bool
	Enabled     bool
	Masked      bool
	Static      bool
	Running     bool
}

// SystemdTimerManager queries systemd for timer units via systemctl commands.
type SystemdTimerManager struct {
	conn shared.Connection
}

func NewSystemdTimerManager(conn shared.Connection) *SystemdTimerManager {
	return &SystemdTimerManager{conn: conn}
}

func (m *SystemdTimerManager) List() ([]*SystemdTimer, error) {
	// Step 1: Get all timer unit files (provides Enabled/Masked/Static/Installed)
	cmdList, err := m.conn.RunCommand("systemctl list-unit-files --type timer --all")
	if err != nil {
		return nil, err
	}

	timers, err := ParseSystemdTimerUnitFiles(cmdList.Stdout)
	if err != nil {
		return nil, err
	}

	// Step 2: Get running state from list-units (provides Running/Description)
	cmdUnits, err := m.conn.RunCommand("systemctl list-units --type timer --all")
	if err != nil {
		return nil, err
	}

	unitStates, err := ParseSystemdTimerListUnits(cmdUnits.Stdout)
	if err != nil {
		return nil, err
	}

	// Step 3: Merge
	for _, timer := range timers {
		unitState, ok := unitStates[timer.Name]
		if !ok {
			continue
		}
		timer.Description = unitState.Description
		timer.Running = unitState.Running
		if !unitState.Installed {
			timer.Installed = false
		}
	}

	return timers, nil
}

func (m *SystemdTimerManager) Get(name string) (*SystemdTimer, error) {
	unit := ensureSystemdTimerUnit(name)
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

	timer := &SystemdTimer{
		Name:        normalizeSystemdTimerName(id),
		Description: props["Description"],
		Installed:   true,
		Running:     props["ActiveState"] == "active",
	}
	applyUnitFileState(&timer.Enabled, &timer.Masked, &timer.Static, props["UnitFileState"])

	return timer, nil
}

// ShowTimerProperties runs systemctl show for timer-specific properties.
func (m *SystemdTimerManager) ShowTimerProperties(name string) (map[string]string, error) {
	unit := ensureSystemdTimerUnit(name)
	cmd, err := m.conn.RunCommand(buildShowPropertyCommand("Unit,OnCalendar,Persistent", unit))
	if err != nil {
		return nil, err
	}

	return parseShowProperties(cmd.Stdout)
}

func ParseSystemdTimerUnitFiles(input io.Reader) ([]*SystemdTimer, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("error executing systemctl list-unit-files --type timer: %v", err)
	}

	var timers []*SystemdTimer
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return timers, nil
	}

	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.Contains(line, "unit files listed.") {
			continue
		}

		timer := &SystemdTimer{
			Name:      normalizeSystemdTimerName(fields[0]),
			Installed: true,
		}
		applyUnitFileState(&timer.Enabled, &timer.Masked, &timer.Static, fields[1])
		timers = append(timers, timer)
	}

	return timers, nil
}

func ParseSystemdTimerListUnits(input io.Reader) (map[string]*SystemdTimer, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("error reading systemctl list-units output: %v", err)
	}

	timers := map[string]*SystemdTimer{}
	matches := SYSTEMD_LIST_UNITS_REGEX.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		unitName := match[1]
		if !strings.HasSuffix(unitName, ".timer") {
			continue
		}

		name := normalizeSystemdTimerName(unitName)
		timers[name] = &SystemdTimer{
			Name:        name,
			Description: strings.TrimSpace(match[5]),
			Running:     match[3] == "active",
			Installed:   match[2] != "not-found",
		}
	}

	return timers, nil
}

func normalizeSystemdTimerName(unit string) string {
	return strings.TrimSuffix(unit, ".timer")
}

func ensureSystemdTimerUnit(name string) string {
	if strings.HasSuffix(name, ".timer") {
		return name
	}
	return name + ".timer"
}
