// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"errors"
	"io"
	"net"
	"slices"
	"strings"

	"github.com/endobit/oui"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

// neti is a helper struct to avoid passing the connection and platform
// as function arguments.
type neti struct {
	connection shared.Connection
	platform   *inventory.Platform
}

// Interfaces returns the network interfaces of the system.
func Interfaces(conn shared.Connection, pf *inventory.Platform) ([]Interface, error) {
	n := &neti{conn, pf}

	if pf.IsFamily(inventory.FAMILY_LINUX) {
		return n.detectLinuxInterfaces()
	}
	if pf.IsFamily(inventory.FAMILY_DARWIN) {
		return n.detectDarwinInterfaces()
	}
	if pf.IsFamily(inventory.FAMILY_WINDOWS) && conn.Capabilities().Has(shared.Capability_File) {
		return n.detectWindowsInterfaces()
	}

	return nil, errors.New("your platform is not supported for the detection of network interfaces")
}

// runCommand is a wrapper around connection.RunCommand that helps execute commands
// and read the standard output for unix and windows systems.
func (n *neti) RunCommand(commandString string) (string, error) {
	if n.platform.IsFamily(inventory.FAMILY_WINDOWS) {
		commandString = powershell.Encode(commandString)
	}
	cmd, err := n.connection.RunCommand(commandString)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

type Interface struct {
	Name        string      `json:"name"`
	MACAddress  string      `json:"mac_address"`
	Vendor      string      `json:"vendor"`
	IPAddresses []IPAddress `json:"ip_addresses"`
	MTU         int         `json:"mtu"`
	Flags       []string    `json:"flags"`
	Active      *bool       `json:"active"`
	Virtual     *bool       `json:"virtual"`
}

// SetMAC is the recommended way to configure the MAC address of
// the interface since it uses it to populate the Vendor field.
func (i *Interface) SetMAC(mac string) {
	if mac != "" {
		i.MACAddress = mac
		i.Vendor = oui.Vendor(mac)
	}
}

// AddOrUpdateInterfaces adds or updates one or many network interfaces
func AddOrUpdateInterfaces(interfaces1, interfaces2 []Interface) (interfaces []Interface) {
	interfaces = interfaces1
	for _, iinterface := range interfaces2 {
		index := findInterface(interfaces, iinterface)

		if index < 0 {
			// not found, add
			log.Trace().Str("name", iinterface.Name).Msg("os.network.interface> add interface")
			interfaces = append(interfaces, iinterface)
			continue
		}

		// found, update
		log.Trace().Str("name", iinterface.Name).Msg("os.network.interface> update interface")
		merged := mergeInterfaces(interfaces[index], iinterface)
		interfaces[index] = merged
	}
	return
}

func mergeInterfaces(i1, i2 Interface) Interface {
	if i1.Name == "" {
		i1.Name = i2.Name
	}
	if i1.MACAddress == "" {
		i1.SetMAC(i2.MACAddress)
	}
	if i1.Vendor == "" {
		i1.Vendor = i2.Vendor
	}
	if i1.MTU == 0 {
		i1.MTU = i2.MTU
	}
	if i1.Active != nil {
		i1.Active = i2.Active
	}
	if i1.Virtual != nil {
		i1.Virtual = i2.Virtual
	}

	for _, flag := range i2.Flags {
		if !slices.Contains(i1.Flags, flag) {
			i1.Flags = append(i1.Flags, flag)
		}
	}

	i1.AddOrUpdateIP(i2.IPAddresses...)

	return i1
}

func findInterface(interfaces []Interface, iinterface Interface) int {
	return slices.IndexFunc(interfaces, func(i Interface) bool {
		return i.Name == iinterface.Name
	})
}

// AddOrUpdateIP adds or updates one or many IPAddresses
func (i *Interface) AddOrUpdateIP(ips ...IPAddress) {
	if i.IPAddresses == nil {
		i.IPAddresses = make([]IPAddress, 0)
	}

	for _, ip := range ips {
		if ip.IP == nil {
			continue
		}

		index := i.findIP(ip.IP)
		if index < 0 {
			// not found, add
			log.Trace().Str("ip", ip.IP.String()).Msg("os.network.interface> add ip")
			i.IPAddresses = append(i.IPAddresses, ip)
			continue
		}

		// found, update
		log.Trace().Str("ip", ip.IP.String()).Msg("os.network.interface> update ip")
		i.mergeIP(index, ip)
	}
}
func (i *Interface) mergeIP(index int, ip IPAddress) {
	merged := mergeIPs(i.IPAddresses[index], ip)
	i.IPAddresses[index] = merged
}
func (i *Interface) findIP(ip net.IP) int {
	return slices.IndexFunc(i.IPAddresses, func(address IPAddress) bool {
		return address.IP.Equal(ip)
	})
}

// mergeIPs takes two IPAddress and merge them together. We give preference
// to the first ip provided.
func mergeIPs(ip1, ip2 IPAddress) IPAddress {
	if ip1.Subnet == "" {
		ip1.Subnet = ip2.Subnet
	}
	if ip1.Gateway == "" {
		ip1.Gateway = ip2.Gateway
	}
	if ip1.CIDR == "" {
		ip1.CIDR = ip2.CIDR
	}
	if ip1.Broadcast == "" {
		ip1.Broadcast = ip2.Broadcast
	}
	return ip1
}
