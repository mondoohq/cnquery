// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"bufio"
	"io"
	"regexp"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

type AixServiceManager struct {
	conn shared.Connection
}

func (s *AixServiceManager) Name() string {
	return "System Resource Controller"
}

func (s *AixServiceManager) List() ([]*Service, error) {
	cmd, err := s.conn.RunCommand("lssrc -a")
	if err != nil {
		return nil, err
	}

	entries := parseLssrc(cmd.Stdout)
	services := make([]*Service, len(entries))
	for i, entry := range entries {
		services[i] = &Service{
			Name:      entry.Subsystem,
			Enabled:   entry.Status == "active",
			Installed: true,
			Running:   entry.Status == "active",
			Type:      "aix",
		}
	}
	return services, nil
}

type lssrcEntry struct {
	Subsystem string
	Group     string
	PID       string
	Status    string
}

var lssrcRegex = regexp.MustCompile(`^\s([\w.-]+)(\s+[\w]+\s+){0,1}([\d]+){0,1}\s+([\w]+)$`)

func parseLssrc(input io.Reader) []lssrcEntry {
	entries := []lssrcEntry{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := lssrcRegex.FindStringSubmatch(line)
		if len(m) == 5 {
			entries = append(entries, lssrcEntry{
				Subsystem: m[1],
				Group:     m[2],
				PID:       m[3],
				Status:    m[4],
			})
		}
	}
	return entries
}
