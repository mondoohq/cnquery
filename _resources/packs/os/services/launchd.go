package services

import (
	"io"
	"io/ioutil"
	"regexp"

	"go.mondoo.com/cnquery/motor/providers/os"
)

var LAUNCHD_REGEX = regexp.MustCompile(`(?m)^\s*([\d-]*)\s+(\d)\s+(.*)$`)

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

// MacOS is using launchd as default service manager
type LaunchDServiceManager struct {
	provider os.OperatingSystemProvider
}

func (s *LaunchDServiceManager) Name() string {
	return "launchd Service Manager"
}

func (s *LaunchDServiceManager) List() ([]*Service, error) {
	c, err := s.provider.RunCommand("launchctl list")
	if err != nil {
		return nil, err
	}
	return ParseServiceLaunchD(c.Stdout)
}
