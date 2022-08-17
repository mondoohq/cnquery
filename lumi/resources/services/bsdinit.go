package services

import (
	"bufio"
	"io"
	"strings"

	"go.mondoo.io/mondoo/motor/providers/os"
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
	provider os.OperatingSystemProvider
}

func (s *BsdInitServiceManager) Name() string {
	return "Bsd Init Service Manager"
}

func (s *BsdInitServiceManager) List() ([]*Service, error) {
	c, err := s.provider.RunCommand("service -e")
	if err != nil {
		return nil, err
	}
	return ParseBsdInit(c.Stdout)
}
