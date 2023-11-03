// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v9/providers/os/fs"
)

func TestParseServiceSystemDUnitFiles(t *testing.T) {
	mock, err := mock.New("./testdata/ubuntu2204.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "ubuntu",
			Family: []string{"ubuntu", "linux"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("systemctl list-unit-files --type service --all")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseServiceSystemDUnitFiles(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 264, len(m), "detected the right amount of services")

	// check first element
	assert.Equal(t, "accounts-daemon.service", m[0].Name, "service name detected")
	assert.Equal(t, true, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "x11-common.service", m[262].Name, "service name detected")
	assert.Equal(t, false, m[262].Running, "service is running")
	assert.Equal(t, false, m[262].Installed, "service is installed")
	assert.Equal(t, "systemd", m[262].Type, "service type is added")

	// check for masked element
	assert.Equal(t, "cryptdisks.service", m[30].Name, "service name detected")
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
