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

type DockerManager struct {
	conn shared.Connection
}

func (d *DockerManager) Name() string {
	return "docker"
}

func (d *DockerManager) IsAvailable() bool {
	cmd, err := d.conn.RunCommand("docker --version")
	if err != nil || cmd.ExitStatus != 0 {
		return false
	}
	return true
}

type dockerContainer struct {
	ID        string `json:"ID"`
	Names     string `json:"Names"`
	Image     string `json:"Image"`
	Status    string `json:"Status"`
	State     string `json:"State"`
	CreatedAt string `json:"CreatedAt"`
	Labels    string `json:"Labels"`
}

func (d *DockerManager) List() ([]*OSContainer, error) {
	// Use docker ps with JSON format
	cmd, err := d.conn.RunCommand("docker ps -a --format '{{json .}}'")
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

	containers := []*OSContainer{}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var dc dockerContainer
		if err := json.Unmarshal([]byte(line), &dc); err != nil {
			continue
		}

		// Parse labels
		labels := make(map[string]string)
		if dc.Labels != "" {
			labelPairs := strings.Split(dc.Labels, ",")
			for _, pair := range labelPairs {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
			}
		}

		// Parse created time
		created, _ := time.Parse(time.RFC3339, dc.CreatedAt)

		container := &OSContainer{
			ID:      dc.ID,
			Name:    strings.TrimPrefix(dc.Names, "/"),
			Image:   dc.Image,
			Status:  dc.Status,
			State:   dc.State,
			Created: created,
			Runtime: "docker",
			Labels:  labels,
		}
		containers = append(containers, container)
	}

	return containers, nil
}
