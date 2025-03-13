// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vmtoolsd

import (
	"fmt"
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/hostname"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

type InstanceIdentifier interface {
	Identify() (Identity, error)
	RawMetadata() (any, error)
}

type Identity struct {
	InstanceID string
	PlatformID string
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(inventory.FAMILY_UNIX) || pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return &CommandInstanceMetadata{
			connection: conn,
			platform:   pf,
		}, nil
	}

	return nil, fmt.Errorf(
		"vmtoolsd id detector is not supported for your asset: %s %s",
		pf.Name, pf.Version,
	)
}

type CommandInstanceMetadata struct {
	connection shared.Connection
	platform   *inventory.Platform
}

func (m *CommandInstanceMetadata) RawMetadata() (any, error) {
	raw := map[string]string{}
	ipaddress, err := m.IPv4()
	if err != nil {
		return raw, err
	}
	raw["ipaddress"] = ipaddress

	hostname, err := m.Hostname()
	if err != nil {
		return raw, err
	}
	raw["hostname"] = hostname

	return raw, nil
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	uuid, err := m.UUID()
	if err != nil {
		return Identity{}, err
	}

	hostname, err := m.Hostname()
	if err != nil {
		return Identity{}, err
	}

	return Identity{
		InstanceID: "//platformid.api.mondoo.app/runtime/vmware/instances/" + uuid,
		PlatformID: "//platformid.api.mondoo.app/runtime/vmware/instances/" + hostname,
	}, nil
}

func (m *CommandInstanceMetadata) UUID() (string, error) {
	uuid, err := m.vmtoolsdGuestInfo("uuid")
	if err == nil && uuid != "" {
		return uuid, nil
	}

	vmid, err := m.vmtoolsdGuestInfo("vmid")
	if err == nil && vmid != "" {
		return vmid, nil
	}

	// try to get this information from the os directly
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		content, err := afero.ReadFile(m.connection.FileSystem(), "/sys/class/dmi/id/product_uuid")
		if err == nil {
			return strings.TrimSpace(string(content)), nil
		}
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		rawUUID, err := m.RunCommand("(Get-WmiObject Win32_ComputerSystemProduct).UUID")
		if err == nil {
			return rawUUID, nil
		}
	}

	return "", errors.New("unable to detect vm uuid")
}

func (m *CommandInstanceMetadata) Hostname() (string, error) {
	name, err := m.vmtoolsdGuestInfo("hostname")
	if err == nil && name != "" {
		return name, nil
	}

	dnsName, err := m.vmtoolsdGuestInfo("dns-name")
	if err == nil && dnsName != "" {
		return dnsName, nil
	}

	// try to get this information from the os directly
	rawHostname, ok := hostname.Hostname(m.connection, m.platform)
	if ok && rawHostname != "" {
		return rawHostname, nil
	}

	return "", errors.New("unable to detect vm hostname")
}

func (m *CommandInstanceMetadata) IPv4() (string, error) {
	ipv4, err := m.vmtoolsdGuestInfo("ip")
	if err == nil && ipv4 != "" {
		return ipv4, nil
	}

	// TODO maybe try to get this information from the os directly

	return "", errors.New("unable to detect ipv4")
}

// RunCommand is a wrapper around connection.RunCommand that helps execute commands
// and read the standard output for unix and windows systems.
func (m *CommandInstanceMetadata) RunCommand(commandString string) (string, error) {
	if m.platform.IsFamily(inventory.FAMILY_WINDOWS) {
		commandString = powershell.Encode(commandString)
	}
	cmd, err := m.connection.RunCommand(commandString)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// vmtoolsdGuestInfo runs vmtoolsd to fetch guest info with the provided key
func (m *CommandInstanceMetadata) vmtoolsdGuestInfo(key string) (string, error) {
	return m.RunCommand(fmt.Sprintf("%s --cmd \"info-get guestinfo.%s", m.vmtoolsd(), key))
}

func (m *CommandInstanceMetadata) vmtoolsd() string {
	if m.platform.IsFamily(inventory.FAMILY_WINDOWS) {
		return `C:\Program Files\VMware\VMware Tools\vmtoolsd.exe`
	}
	return "vmtoolsd"
}
