// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/fs"
)

func TestSystemDExtractDescription(t *testing.T) {
	statusout := `
  ● avahi-daemon.service - Avahi mDNS/DNS-SD Stack
     Loaded: loaded (/lib/systemd/system/avahi-daemon.service; enabled; vendor preset: enabled)
     Active: active (running) since Fri 2023-11-03 06:26:15 CET; 2h 59min ago
TriggeredBy: ● avahi-daemon.socket
   Main PID: 1219 (avahi-daemon)
     Status: "avahi-daemon 0.8 starting up."
      Tasks: 2 (limit: 38013)
     Memory: 1.4M
        CPU: 1.173s
     CGroup: /system.slice/avahi-daemon.service
             ├─1219 "avahi-daemon: running [mondoopad.local]"
             └─1297 "avahi-daemon: chroot helper"

Nov 03 06:46:32 mondoopad avahi-daemon[1219]: Interface wlp0s20f3.IPv6 no longer relevant for mDNS.
Nov 03 06:46:32 mondoopad avahi-daemon[1219]: Withdrawing address record for 192.168.178.32 on wlp0s20f3.
Nov 03 06:46:32 mondoopad avahi-daemon[1219]: Leaving mDNS multicast group on interface wlp0s20f3.IPv4 with address 192.168.178.32.
Nov 03 06:46:32 mondoopad avahi-daemon[1219]: Interface wlp0s20f3.IPv4 no longer relevant for mDNS.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: Joining mDNS multicast group on interface wlp0s20f3.IPv6 with address fe80::80d3:b6d1:14e3:56e1.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: New relevant interface wlp0s20f3.IPv6 for mDNS.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: Registering new address record for fe80::80d3:b6d1:14e3:56e1 on wlp0s20f3.*.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: Joining mDNS multicast group on interface wlp0s20f3.IPv4 with address 192.168.178.32.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: New relevant interface wlp0s20f3.IPv4 for mDNS.
Nov 03 06:51:44 mondoopad avahi-daemon[1219]: Registering new address record for 192.168.178.32 on wlp0s20f3.IPv4.
  `
	description := SystemDExtractDescription(statusout)

	assert.Equal(t, description, "Avahi mDNS/DNS-SD Stack")
}

func TestParseServiceSystemDUnitFiles(t *testing.T) {
	mock, err := mock.New(0, "./testdata/ubuntu2204.toml", &inventory.Asset{
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
	assert.Equal(t, "accounts-daemon", m[0].Name, "service name detected")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "x11-common", m[262].Name, "service name detected")
	assert.Equal(t, "systemd", m[262].Type, "service type is added")

	// check for masked element
	assert.Equal(t, "cryptdisks", m[30].Name, "service name detected")
	assert.Equal(t, true, m[30].Masked, "service is masked")
}

func TestParseServiceSystemDUnitFilesPhoton(t *testing.T) {
	mock, err := mock.New(0, "./testdata/photon.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "photon",
			Family: []string{"redhat", "linux"},
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
	assert.Equal(t, 138, len(m), "detected the right amount of services")

	// check first element
	assert.Equal(t, "autovt@", m[0].Name, "service name detected")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "vmtoolsd", m[136].Name, "service name detected")
	assert.Equal(t, "systemd", m[136].Type, "service type is added")

	// check for masked element
	assert.Equal(t, "dracut-pre-udev", m[30].Name, "service name detected")
	assert.Equal(t, false, m[30].Masked, "service is not masked")
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
