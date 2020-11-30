package services

import (
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"go.mondoo.io/mondoo/motor"
)

var (
	SYSTEMD_LIST_UNITS_REGEX = regexp.MustCompile(`(?m)^(?:[^\S\n]{2}|●[^\S\n]|)(\S+)(?:[^\S\n])+(loaded|not-found|masked)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(.+)$`)
)

// a line may be prefixed with nothing, whitespace or a dot
func ParseServiceSystemDUnitFiles(input io.Reader) ([]*Service, error) {
	var services []*Service
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	m := SYSTEMD_LIST_UNITS_REGEX.FindAllStringSubmatch(string(content), -1)
	for i := range m {
		name := m[i][1]
		name = strings.Replace(name, ".service", "", 1)

		s := &Service{
			Name:      name,
			Installed: m[i][2] == "loaded",
			Running:   m[i][3] == "active",
			// TODO: we may need to revist the enabled state
			Enabled:     m[i][2] == "loaded",
			Masked:      m[i][2] == "masked",
			Description: m[i][5],
			Type:        "systemd",
		}
		services = append(services, s)
	}
	return services, nil
}

// Newer linux systems use systemd as service manager
type SystemDServiceManager struct {
	motor *motor.Motor
}

func (s *SystemDServiceManager) Name() string {
	return "systemd Service Manager"
}

func (s *SystemDServiceManager) List() ([]*Service, error) {
	c, err := s.motor.Transport.RunCommand("systemctl --all list-units --type service")
	if err != nil {
		return nil, err
	}
	return ParseServiceSystemDUnitFiles(c.Stdout)
}
