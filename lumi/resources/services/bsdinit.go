package services

import (
	"bufio"
	"io"
	"strings"

	motor "go.mondoo.io/mondoo/motor/motoros"
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
	motor *motor.Motor
}

func (s *BsdInitServiceManager) Name() string {
	return "Bsd Init Service Manager"
}

func (s *BsdInitServiceManager) Service(name string) (*Service, error) {
	services, err := s.List()
	if err != nil {
		return nil, err
	}

	return findService(services, name)
}

func (s *BsdInitServiceManager) List() ([]*Service, error) {
	c, err := s.motor.Transport.RunCommand("service -e")
	if err != nil {
		return nil, err
	}
	return ParseBsdInit(c.Stdout)
}
