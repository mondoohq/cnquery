// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package hetznercloud

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"gopkg.in/yaml.v3"
)

const (
	metadataSvcURL = "http://169.254.169.254/hetzner/v1/metadata"
)

func MondooHetznerInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/hetzner/instances/" + instanceID
}

type Identity struct {
	InstanceID string
	Hostname   string
	Region     string
}

type InstanceIdentifier interface {
	Identify() (Identity, error)
	RawMetadata() (any, error)
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(inventory.FAMILY_UNIX) {
		return &commandInstanceMetadata{conn, pf}, nil
	}
	return nil, fmt.Errorf(
		"hetzner cloud id detector is not supported for your asset: %s %s",
		pf.Name, pf.Version,
	)
}

// hetznerMetadata represents the YAML structure returned by the Hetzner metadata service
type hetznerMetadata struct {
	InstanceID       int    `yaml:"instance-id"`
	Hostname         string `yaml:"hostname"`
	Region           string `yaml:"region"`
	AvailabilityZone string `yaml:"availability-zone"`
	LocalIPv4        string `yaml:"local-ipv4"`
	PublicIPv4       string `yaml:"public-ipv4"`
}

type commandInstanceMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (m *commandInstanceMetadata) RawMetadata() (any, error) {
	data, err := m.fetchMetadata()
	if err != nil {
		return nil, err
	}

	var rawMap map[string]any
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return nil, err
	}
	return rawMap, nil
}

func (m *commandInstanceMetadata) Identify() (Identity, error) {
	data, err := m.fetchMetadata()
	if err != nil {
		return Identity{}, err
	}

	md := hetznerMetadata{}
	if err := yaml.Unmarshal(data, &md); err != nil {
		return Identity{}, fmt.Errorf("failed to decode Hetzner metadata: %w", err)
	}

	if md.InstanceID == 0 {
		return Identity{}, errors.New("hetzner metadata did not contain an instance-id")
	}

	return Identity{
		InstanceID: MondooHetznerInstanceID(fmt.Sprintf("%d", md.InstanceID)),
		Hostname:   md.Hostname,
		Region:     md.Region,
	}, nil
}

func (m *commandInstanceMetadata) fetchMetadata() ([]byte, error) {
	cmdStr := fmt.Sprintf("curl --retry 3 --retry-delay 1 --connect-timeout 1 --retry-max-time 5 --max-time 10 --noproxy '*' %s", metadataSvcURL)
	cmd, err := m.conn.RunCommand(cmdStr)
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		return nil, fmt.Errorf("hetzner metadata request failed with exit status %d", cmd.ExitStatus)
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, errors.New("empty response from Hetzner metadata service")
	}

	return []byte(content), nil
}
