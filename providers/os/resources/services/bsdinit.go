// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"bufio"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

func ParseBsdInit(input io.Reader) ([]*Service, error) {
	var services []*Service
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		services = append(services, &Service{
			Name:      strings.TrimSpace(line),
			Enabled:   true,
			Installed: true,
			Running:   true,
			Type:      "bsd",
		})
	}
	return services, nil
}

type BsdInitServiceManager struct {
	conn shared.Connection
}

func (s *BsdInitServiceManager) Name() string {
	return "Bsd Init Service Manager"
}

func (s *BsdInitServiceManager) List() ([]*Service, error) {
	c, err := s.conn.RunCommand("service -e")
	if err != nil {
		return nil, err
	}
	return ParseBsdInit(c.Stdout)
}
