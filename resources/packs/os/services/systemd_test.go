package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/providers/fs"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestParseServiceSystemDUnitFiles(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/debian.toml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("systemctl --all list-units --type service")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseServiceSystemDUnitFiles(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 102, len(m), "detected the right amount of services")

	// check first element
	assert.Equal(t, "auditd", m[0].Name, "service name detected")
	assert.Equal(t, true, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "ypxfrd", m[101].Name, "service name detected")
	assert.Equal(t, false, m[101].Running, "service is running")
	assert.Equal(t, false, m[101].Installed, "service is installed")
	assert.Equal(t, "systemd", m[101].Type, "service type is added")

	// check for masked element
	assert.Equal(t, "nfs-server", m[30].Name, "service name detected")
	assert.Equal(t, true, m[30].Masked, "service is masked")
}

func TestSystemdFS(t *testing.T) {
	s := SystemdFSServiceManager{
		Fs: fs.NewMountedFs("testdata/systemd"),
	}

	services, err := s.List()
	require.NoError(t, err)
	servicesMap := map[string]*Service{}
	for _, svc := range services {
		servicesMap[svc.Name] = svc
	}

	assert.NotContains(t, servicesMap, "default")
	assert.NotContains(t, servicesMap, "default.target")
	assert.NotContains(t, servicesMap, "not-enabled")
	assert.Contains(t, servicesMap, "aliased")
	assert.Equal(t, &Service{
		Name:        "aliased",
		Type:        "service",
		Description: "Aliased Service",
		State:       ServiceUnknown,
		Installed:   true,
		Enabled:     true,
	}, servicesMap["aliased"])
	assert.Contains(t, servicesMap, "aliased-wants")
	assert.Contains(t, servicesMap, "aliased-requires")
	assert.Contains(t, servicesMap, "aliased-missing")
	assert.Equal(t, &Service{
		Name:      "aliased-missing",
		Type:      "service",
		State:     ServiceUnknown,
		Installed: false,
		Enabled:   false,
	}, servicesMap["aliased-missing"])
	assert.Contains(t, servicesMap, "intermediate-dep-want")
	assert.Contains(t, servicesMap, "intermediate-dep-require")
	assert.Contains(t, servicesMap, "masked")
	assert.Equal(t, &Service{
		Name:      "masked",
		Type:      "service",
		State:     ServiceUnknown,
		Installed: true,
		Enabled:   true,
		Masked:    true,
	}, servicesMap["masked"])
	assert.Contains(t, servicesMap, "implicit-socket")
	assert.Equal(t, &Service{
		Name:        "implicit-socket",
		Type:        "service",
		Description: "Implicit Socket Service",
		State:       ServiceUnknown,
		Installed:   true,
		Enabled:     true,
	}, servicesMap["implicit-socket"])
	assert.Contains(t, servicesMap, "explicit-socket-service")
	assert.Equal(t, &Service{
		Name:        "explicit-socket-service",
		Type:        "service",
		Description: "Explicit Socket Service",
		State:       ServiceUnknown,
		Installed:   true,
		Enabled:     true,
	}, servicesMap["explicit-socket-service"])

	// Relative path symlink
	assert.Contains(t, servicesMap, "display-manager")
	assert.Equal(t, &Service{
		Name:      "display-manager",
		Type:      "service",
		State:     ServiceUnknown,
		Installed: false,
		Enabled:   false,
		Masked:    false,
	}, servicesMap["display-manager"])

	// Absolute path symlink
	assert.Contains(t, servicesMap, "sshd")
	assert.Equal(t, &Service{
		Name:        "sshd",
		Description: "OpenBSD Secure Shell server",
		Type:        "service",
		State:       ServiceUnknown,
		Installed:   true,
		Enabled:     true,
		Masked:      false,
	}, servicesMap["sshd"])
}
