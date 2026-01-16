// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package containers

import (
	"encoding/json"
	"io"
	"time"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

type PodmanManager struct {
	conn shared.Connection
}

func (p *PodmanManager) Name() string {
	return "podman"
}

func (p *PodmanManager) IsAvailable() bool {
	cmd, err := p.conn.RunCommand("podman --version")
	if err != nil || cmd.ExitStatus != 0 {
		return false
	}
	return true
}

type podmanContainer struct {
	ID         string            `json:"Id"`
	Names      []string          `json:"Names"`
	Image      string            `json:"Image"`
	Status     string            `json:"Status"`
	State      string            `json:"State"`
	Created    string            `json:"Created"`
	Labels     map[string]string `json:"Labels"`
}

func (p *PodmanManager) List() ([]*OSContainer, error) {
	// Use podman ps with JSON format
	cmd, err := p.conn.RunCommand("podman ps -a --format json")
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

	var podmanContainers []podmanContainer
	if err := json.Unmarshal(data, &podmanContainers); err != nil {
		return nil, err
	}

	containers := []*OSContainer{}
	for _, pc := range podmanContainers {
		// Get the first name
		name := ""
		if len(pc.Names) > 0 {
			name = pc.Names[0]
		}

		// Parse created time
		created, _ := time.Parse(time.RFC3339, pc.Created)

		container := &OSContainer{
			ID:      pc.ID,
			Name:    name,
			Image:   pc.Image,
			Status:  pc.Status,
			State:   pc.State,
			Created: created,
			Runtime: "podman",
			Labels:  pc.Labels,
		}
		containers = append(containers, container)
	}

	return containers, nil
}
