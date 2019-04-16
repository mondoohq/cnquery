package services

import (
	"io"
	"io/ioutil"
	"regexp"
)

type Service struct {
	Name        string
	Description string
	State       string
	Type        string
	Installed   bool
	Running     bool
	Enabled     bool
}

var (
	SYSTEMD_LIST_UNITS_REGEX = regexp.MustCompile(`(?m)^(?:[^\S\n]{2}|●[^\S\n])(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(.+)$`)
	LAUNCHD_REGEX            = regexp.MustCompile(`(?m)^\s*([\d-]*)\s+(\d)\s+(.*)$`)
)

// ^(?:[^\S\n]*[●]*)+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(.+)$
func ParseServiceSystemDUnitFiles(input io.Reader) ([]*Service, error) {
	var services []*Service
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	m := SYSTEMD_LIST_UNITS_REGEX.FindAllStringSubmatch(string(content), -1)
	for i := range m {
		// ignore header
		if i == 0 {
			continue
		}
		s := &Service{
			Name:        m[i][1],
			Installed:   m[i][2] == "loaded",
			Running:     m[i][3] == "active",
			Description: m[i][5],
			Type:        "systemd",
		}
		services = append(services, s)
	}
	return services, nil
}

// PID: pid of process
// Status: last know exit code
// ^\s*([\d-]*)\s+(\d)\s+(.*)$
func ParseServiceLaunchD(input io.Reader) ([]*Service, error) {
	var services []*Service
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	m := LAUNCHD_REGEX.FindAllStringSubmatch(string(content), -1)
	for i := range m {
		s := &Service{
			Name:      m[i][3],
			Enabled:   true,
			Installed: true,
			Running:   m[i][1] != "-",
			Type:      "launchd",
		}
		services = append(services, s)
	}
	return services, nil
}
