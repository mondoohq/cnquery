// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"bufio"
	"io"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

type AlpineOpenrcServiceManager struct {
	conn shared.Connection
}

func (s *AlpineOpenrcServiceManager) Name() string {
	return "OpenRC Init Service Manager"
}

func (s *AlpineOpenrcServiceManager) List() ([]*Service, error) {
	// retrieve service list by retrieving all files
	var services []*Service

	f, err := s.conn.FileSystem().Open("/etc/init.d")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	files, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	// retrieve service status from running systems
	serviceStatusMap := map[string]bool{}
	if s.conn.Capabilities().Has(shared.Capability_RunCommand) {
		// check if the rc-status command exits, if not no service is running
		cmd, err := s.conn.RunCommand("which rc-status")
		if err != nil {
			return nil, err
		}

		// if the rc-status command is installed
		cmdOut, _ := io.ReadAll(cmd.Stdout)
		if string(cmdOut) != "" {
			cmd, err := s.conn.RunCommand("rc-status -s")
			if err != nil {
				return nil, err
			}

			serviceStatusMap, err = ParseOpenRCServiceStatus(cmd.Stdout)
			if err != nil {
				return nil, err
			}
		}
	}

	// check for services in runlevel
	runlevel, err := determineOpenRcRunlevel(s.conn.FileSystem())
	if err != nil {
		return nil, err
	}

	// build up service objects
	for i := range files {
		serviceFile := files[i]
		name := serviceFile.Name()
		services = append(services, &Service{
			Name:      name,
			Enabled:   runlevel[name], // read status from rc
			Installed: true,
			Running:   serviceStatusMap[name], // read from status from rc-status command
			Type:      "openrc",
		})
	}

	return services, nil
}

var OPENRC_SERVICE_STARTED = regexp.MustCompile(`^([a-zA-Z-\d]+)\s+\[\s*(stopped|started)\s*\]$`)

func ParseOpenRCServiceStatus(input io.Reader) (map[string]bool, error) {
	status := map[string]bool{}

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := OPENRC_SERVICE_STARTED.FindStringSubmatch(line)
		if len(m) == 3 {
			status[m[1]] = (m[2] == "started")
		}
	}
	return status, nil
}

func determineOpenRcRunlevel(fs afero.Fs) (map[string]bool, error) {
	activated := map[string]bool{}
	runlevelRoot := "/etc/runlevels/"

	afutil := afero.Afero{Fs: fs}
	ok, err := afutil.Exists(runlevelRoot)
	if err != nil {
		return nil, err
	}

	if ok {
		f, err := fs.Open(runlevelRoot)
		if err != nil {
			return nil, err
		}

		// iterate over level
		levels, err := f.Readdirnames(-1)
		if err != nil {
			return nil, err
		}

		for i := range levels {
			level := levels[i]

			levelF, err := fs.Open(filepath.Join(runlevelRoot, level))
			if err != nil {
				return nil, err
			}

			serviceNames, err := levelF.Readdirnames(-1)
			if err != nil {
				levelF.Close()
				return nil, err
			}
			levelF.Close()

			for j := range serviceNames {
				activated[serviceNames[j]] = true
			}
		}
	}

	return activated, nil
}
