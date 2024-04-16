// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"fmt"
	"io"

	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/packages"
)

const (
	SuseOSUpdateFormat = "suse"
)

type SuseUpdateManager struct {
	conn shared.Connection
}

func (sum *SuseUpdateManager) Name() string {
	return "Suse Update Manager"
}

func (sum *SuseUpdateManager) Format() string {
	return SuseOSUpdateFormat
}

func (sum *SuseUpdateManager) List() ([]OperatingSystemUpdate, error) {
	cmd, err := sum.conn.RunCommand("zypper -n --xmlout list-updates -t patch")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return ParseZypperPatches(cmd.Stdout)
}

// ParseZypperPatches reads the operating system patches for Suse
func ParseZypperPatches(input io.Reader) ([]OperatingSystemUpdate, error) {
	zypper, err := packages.ParseZypper(input)
	if err != nil {
		return nil, err
	}

	var updates []OperatingSystemUpdate
	// filter for kind patch
	for _, u := range zypper.Updates {
		if u.Kind != "patch" {
			continue
		}

		restart := false
		if u.Restart == "true" {
			restart = true
		}

		updates = append(updates, OperatingSystemUpdate{
			ID:          u.Edition,
			Name:        u.Name,
			Severity:    u.Severity,
			Restart:     restart,
			Category:    u.Category,
			Description: u.Description,
			Format:      SuseOSUpdateFormat,
		})
	}

	return updates, nil
}
