// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// detectLinuxInterfaces detects network interfaces on Linux.
func (n *neti) detectLinuxInterfaces() ([]Interface, error) {
	var errs []error
	interfaces := []Interface{}

	// List of detectors that collect network interfaces, we stop executing them as
	// soon as one of them succeeds collecting all the information
	detectors := []func() ([]Interface, error){
		n.getLinuxCmdInterfaces,
		n.getLinuxSysfsInterfaces,
	}
	for _, detectFn := range detectors {
		detectedInterfaces, err := detectFn()
		if err == nil && len(detectedInterfaces) != 0 {
			interfaces = AddOrUpdateInterfaces(interfaces, detectedInterfaces)
			break
		}
		log.Debug().Err(err).
			Msg("os.network.interface> unable to detect network interfaces")
		errs = append(errs, err)
	}

	// List of enrichment functions that collect additional information that we
	// couldn't gather in the detectors.
	enrichments := []func() ([]Interface, error){
		n.getLinuxIPv4GatewayDetails,
		n.getLinuxIPv6GatewayDetails,
	}

	for _, detectFn := range enrichments {
		detectedInterfaces, err := detectFn()
		if err != nil {
			log.Debug().Err(err).
				Msg("os.network.interface> unable to detect network interfaces")
			errs = append(errs, err)
			continue
		}
		interfaces = AddOrUpdateInterfaces(interfaces, detectedInterfaces)
	}

	if len(interfaces) == 0 {
		return interfaces, errors.Join(errs...)
	}

	return interfaces, nil
}

func (n *neti) getLinuxIPv4GatewayDetails() (interfaces []Interface, err error) {
	output, err := n.RunCommand("ip route show")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// we are looking for lines like this one
		//
		// default via 172.31.16.1 dev enX0
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) > 4 && isDefaultRoute(fields[0]) {
			interfaces = append(interfaces, Interface{
				Name: fields[4],
				enrichments: func(in *Interface) {
					for i := range in.IPAddresses {
						if version, ok := in.IPAddresses[i].Version(); ok && version == IPv4 {
							in.IPAddresses[i].Gateway = fields[2]
						}
					}
				},
			})
		}
	}
	return
}

func (n *neti) getLinuxIPv6GatewayDetails() (interfaces []Interface, err error) {
	output, err := n.RunCommand("ip -6 route show")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// we are looking for lines like this one
		//
		// default via 2001:db8::1 dev enX1
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) > 4 && isDefaultRoute(fields[0]) {
			interfaces = append(interfaces, Interface{
				Name: fields[4],
				enrichments: func(in *Interface) {
					for i := range in.IPAddresses {
						if version, ok := in.IPAddresses[i].Version(); ok && version == IPv6 {
							in.IPAddresses[i].Gateway = fields[2]
						}
					}
				},
			})
		}
	}
	return
}

func (n *neti) getLinuxSysfsInterfaces() (interfaces []Interface, err error) {
	dirEntries, err := afero.ReadDir(n.connection.FileSystem(), "/sys/class/net")
	if err != nil {
		return nil, err
	}

	log.Debug().Int("dir_entries", len(dirEntries)).Msg("os.network.interface> read /sys/class/net")

	for _, entry := range dirEntries {
		if !entry.IsDir() {
			log.Trace().
				Str("name", filepath.Join("/sys/class/net", entry.Name())).
				Msg("os.network.interfaces> not a directory, skipping")
			continue
		}

		ifaceName := entry.Name()
		iinterface := Interface{Name: ifaceName}

		// Read MAC Address
		macAddress, err := afero.ReadFile(
			n.connection.FileSystem(),
			filepath.Join("/sys/class/net/", ifaceName, "address"),
		)
		if err == nil {
			iinterface.SetMAC(strings.TrimSpace(string(macAddress)))
		}

		// Read MTU
		mtu, err := afero.ReadFile(
			n.connection.FileSystem(),
			filepath.Join("/sys/class/net/", ifaceName, "mtu"),
		)
		if err == nil {
			iinterface.MTU = parseInt(strings.TrimSpace(string(mtu)))
		}

		// Read device type
		deviceType, err := afero.ReadFile(
			n.connection.FileSystem(),
			filepath.Join("/sys/class/net/", ifaceName, "device/devtype"),
		)
		if err == nil {
			iinterface.Virtual = isVirtualDevice(string(deviceType))
		}

		// Read Flags
		flags, err := afero.ReadFile(
			n.connection.FileSystem(),
			filepath.Join("/sys/class/net/", ifaceName, "flags"),
		)
		if err == nil {
			iinterface.Flags = parseHexFlags(strings.TrimPrefix(string(flags), "0x"))
		}

		// Read Status
		operstate, err := afero.ReadFile(
			n.connection.FileSystem(),
			filepath.Join("/sys/class/net/", ifaceName, "operstate"),
		)
		if err == nil {
			switch strings.TrimSpace(strings.ToLower(string(operstate))) {
			case "up":
				iinterface.Active = convert.ToPtr(true)
			case "down":
				iinterface.Active = convert.ToPtr(false)
			}
		}

		// TODO we could fetch statistics from here, like `tx_queue_len` or `rx_bytes` and `tx_bytes`
		// iinterface.Statistics = getLinuxInterfaceStats(ifaceName)

		interfaces = append(interfaces, iinterface)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.sys_class_net_fs").
		Msg("os.network.interfaces> discovered")

	return
}

func parseHexFlags(hexStr string) []string {
	flagsMap := map[int]string{
		0x1:    "UP",
		0x2:    "BROADCAST",
		0x4:    "DEBUG",
		0x8:    "LOOPBACK",
		0x10:   "POINTOPOINT",
		0x20:   "NOTRAILERS",
		0x40:   "RUNNING",
		0x80:   "NOARP",
		0x100:  "PROMISC",
		0x200:  "ALLMULTI",
		0x400:  "MASTER",
		0x800:  "SLAVE",
		0x1000: "MULTICAST",
		0x2000: "PORTSEL",
		0x4000: "AUTOMEDIA",
		0x8000: "DYNAMIC",
	}
	flagsInt, err := strconv.ParseInt(hexStr, 16, 32)
	if err != nil {
		return []string{}
	}
	var flags []string
	for bit, name := range flagsMap {
		if int(flagsInt)&bit != 0 {
			flags = append(flags, name)
		}
	}
	return flags
}

func (n *neti) getLinuxCmdInterfaces() ([]Interface, error) {
	output, err := n.RunCommand("ip addr show")
	if err != nil {
		return nil, err
	}

	var (
		interfaces     = []Interface{}
		scanner        = bufio.NewScanner(strings.NewReader(string(output)))
		interfaceRegex = regexp.MustCompile(`^\d+: ([^:]+): <([^>]+)> mtu (\d+)`)
		macRegex       = regexp.MustCompile(`link/ether ([0-9a-fA-F:]+) `)
		ipRegex        = regexp.MustCompile(`inet ([0-9\.]+)/([0-9]+)`)
		ip6Regex       = regexp.MustCompile(`inet6 ([0-9a-fA-F:]+)/([0-9]+) scope (global|link|host)`)
		// TODO @afiune we could add additional information to the interface struct like
		//  * `altname` (alternative name)
		//  * `metric` (priority or cost of a network interface)
	)

	var currentInterface *Interface
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if matches := interfaceRegex.FindStringSubmatch(line); matches != nil {
			if currentInterface != nil {
				interfaces = append(interfaces, *currentInterface)
			}
			mtu := parseInt(matches[3])
			flags := strings.Split(matches[2], ",")
			active := strings.Contains(matches[2], "UP")
			virtual := strings.HasPrefix(matches[1], "veth") || strings.HasPrefix(matches[1], "virbr")
			currentInterface = &Interface{
				Name:        matches[1],
				MTU:         mtu,
				Flags:       flags,
				Active:      &active,
				Virtual:     &virtual,
				IPAddresses: []IPAddress{},
			}
		} else if currentInterface != nil {
			if matches := macRegex.FindStringSubmatch(line); matches != nil {
				// Match MAC address
				currentInterface.SetMAC(matches[1])
			} else if matches := ipRegex.FindStringSubmatch(line); matches != nil {
				// Match IPv4 address
				currentInterface.AddOrUpdateIP(NewIPv4WithPrefixLength(
					matches[1],
					parseInt(matches[2]),
				))
			} else if matches := ip6Regex.FindStringSubmatch(line); matches != nil {
				// Match IPv6 address
				ip := NewIPv6WithPrefixLength(matches[1], parseInt(matches[2]))
				currentInterface.AddOrUpdateIP(ip)
			}
		}
	}

	if currentInterface != nil {
		interfaces = append(interfaces, *currentInterface)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.ip_addr_show").
		Msg("os.network.interfaces> discovered")
	return interfaces, nil
}

func isVirtualDevice(deviceType string) *bool {
	switch deviceType {
	// The device is a VIF (Xen virtual interface).
	case "vif":
		return convert.ToPtr(true)

	// The device is a TUN (network layer) virtual network interface.
	case "tun":
		return convert.ToPtr(true)

	// The device is a TAP (data link layer) virtual network interface.
	case "tap":
		return convert.ToPtr(true)

	// The device is a MACVLAN virtual network interface.
	case "macvlan":
		return convert.ToPtr(true)

	// The device is an IPVLan virtual network interface.
	case "ipvlan":
		return convert.ToPtr(true)

	// The device is a Virtual Ethernet (veth) interface.
	case "veth":
		return convert.ToPtr(true)

	// The device is a network bridge.
	case "bridge":
		return convert.ToPtr(true)

	// The device is a bonded network interface (multiple NICs bonded together).
	case "bond":
		return convert.ToPtr(true)

	// The device is a VXLAN virtual network tunnel interface.
	case "vxlan":
		return convert.ToPtr(true)

	// The device is a GRE (Generic Routing Encapsulation) tunnel interface.
	case "gre":
		return convert.ToPtr(false)

	// The device is a GRE TAP (Layer 2 GRE) tunnel interface.
	case "gretap":
		return convert.ToPtr(true)

	// The device is an IPv6 GRE tunnel interface.
	case "ip6gre":
		return convert.ToPtr(false)

	// The device is an IPv6 GRE TAP tunnel interface.
	case "ip6gretap":
		return convert.ToPtr(true)

	// The device is an IPv6-in-IPv4 tunnel.
	case "sit":
		return convert.ToPtr(false)

	// The device is an IPv4-in-IPv4 tunnel.
	case "ipip":
		return convert.ToPtr(false)

	// The device is a WireGuard VPN interface.
	case "wireguard":
		return convert.ToPtr(true)

	// The device is a Point-to-Point Protocol (PPP) interface.
	case "ppp":
		return convert.ToPtr(false)

	// The device is an XFRM (IPsec transform) interface.
	case "xfrm":
		return convert.ToPtr(false)

	}
	return nil
}
