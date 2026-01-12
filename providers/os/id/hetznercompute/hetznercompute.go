// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hetznercompute

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/powershell"
)

// Hetzner Cloud metadata service endpoint
const metadataURLPath = "http://169.254.169.254/hetzner/v1/metadata"

type Identity struct {
	InstanceName string
	InstanceID   string
}

type InstanceIdentifier interface {
	Identify() (Identity, error)
	RawMetadata() (any, error)
}

type commandInstanceMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(inventory.FAMILY_UNIX) || pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return &commandInstanceMetadata{conn, pf}, nil
	}
	return nil, errors.New("hetzner compute id detector is not supported for your asset: " + pf.Name + " " + pf.Version)
}

func (m *commandInstanceMetadata) Identify() (Identity, error) {
	metadata, err := m.metadataDocument()
	if err != nil {
		return Identity{}, err
	}

	instanceID := ""
	instanceName := ""

	if id, ok := metadata["instance-id"]; ok {
		instanceID = fmt.Sprintf("%v", id)
	} else {
		return Identity{}, errors.New("instance-id not found in metadata")
	}

	if hostname, ok := metadata["hostname"]; ok {
		instanceName = fmt.Sprintf("%v", hostname)
	}

	mondooInstanceID := "//platformid.api.mondoo.app/runtime/hetzner/compute/v1/instances/" + instanceID

	return Identity{
		InstanceID:   mondooInstanceID,
		InstanceName: instanceName,
	}, nil
}

func (m *commandInstanceMetadata) RawMetadata() (any, error) {
	return m.metadataDocument()
}

func (m *commandInstanceMetadata) metadataDocument() (map[string]any, error) {
	var (
		cmd *shared.Command
		err error
	)

	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		cmd, err = m.conn.RunCommand("curl --noproxy '*' --retry 2 --connect-timeout 1 --max-time 3 " + metadataURLPath)
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		script := fmt.Sprintf(`Invoke-RestMethod -TimeoutSec 3 -Uri "%s" -UseBasicParsing | ConvertTo-Json`, metadataURLPath)
		cmd, err = m.conn.RunCommand(powershell.Encode(script))
	default:
		err = errors.New("your platform is not supported by hetzner metadata identifier resource")
	}

	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	// Hetzner metadata can be returned as text (key: value) or JSON
	// Try to parse as JSON first
	metadata := make(map[string]any)
	if err := json.Unmarshal(data, &metadata); err == nil {
		// Verify it's actually a map/object, not just a JSON-encoded string
		if len(metadata) > 0 {
			return metadata, nil
		}
	}

	// If JSON parsing fails or resulted in empty map, try parsing as a JSON string first
	// (PowerShell's ConvertTo-Json might have encoded the text as a JSON string)
	var jsonString string
	if err := json.Unmarshal(data, &jsonString); err == nil {
		// It's a JSON string, use it as text
		text := strings.TrimSpace(jsonString)
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				metadata[key] = value
			}
		}
	} else {
		// Not JSON at all, parse as raw text format (key: value pairs)
		text := strings.TrimSpace(string(data))
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				metadata[key] = value
			}
		}
	}

	if len(metadata) == 0 {
		return nil, errors.New("metadata service returned empty or invalid response")
	}

	return metadata, nil
}
