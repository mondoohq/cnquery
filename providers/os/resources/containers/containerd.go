// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package containers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

type ContainerdManager struct {
	conn shared.Connection
}

func (c *ContainerdManager) Name() string {
	return "containerd"
}

func (c *ContainerdManager) IsAvailable() bool {
	// Try ctr command (containerd CLI)
	cmd, err := c.conn.RunCommand("ctr version")
	if err != nil || cmd.ExitStatus != 0 {
		// Also try nerdctl (Docker-compatible CLI for containerd)
		cmd, err = c.conn.RunCommand("nerdctl version")
		if err != nil || cmd.ExitStatus != 0 {
			return false
		}
	}
	return true
}

type containerdContainer struct {
	ID     string `json:"id"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

func (c *ContainerdManager) List() ([]*OSContainer, error) {
	// Try nerdctl first (has better JSON output)
	cmd, err := c.conn.RunCommand("nerdctl ps -a --format json")
	if err == nil && cmd.ExitStatus == 0 {
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return nil, err
		}
		return c.parseNerdctlOutput(string(data))
	}

	// Fall back to ctr
	cmd, err = c.conn.RunCommand("ctr containers list")
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	return c.parseCtrOutput(string(data))
}

func (c *ContainerdManager) parseNerdctlOutput(output string) ([]*OSContainer, error) {
	containers := []*OSContainer{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var nc containerdContainer
		if err := json.Unmarshal([]byte(line), &nc); err != nil {
			continue
		}

		container := &OSContainer{
			ID:      nc.ID,
			Name:    nc.ID, // containerd doesn't have names like Docker
			Image:   nc.Image,
			Status:  nc.Status,
			State:   nc.Status,
			Created: time.Time{}, // Would need additional API calls
			Runtime: "containerd",
			Labels:  make(map[string]string),
		}
		containers = append(containers, container)
	}

	return containers, nil
}

func (c *ContainerdManager) parseCtrOutput(output string) ([]*OSContainer, error) {
	containers := []*OSContainer{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Skip header line
	if len(lines) < 2 {
		return containers, nil
	}

	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		container := &OSContainer{
			ID:      fields[0],
			Name:    fields[0], // ctr doesn't show names
			Image:   fields[1],
			Status:  "unknown",
			State:   "unknown",
			Created: time.Time{},
			Runtime: "containerd",
			Labels:  make(map[string]string),
		}
		containers = append(containers, container)
	}

	return containers, nil
}
