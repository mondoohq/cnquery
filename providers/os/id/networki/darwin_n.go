// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"howett.net/plist"
)

// detectDarwinInterfaces detects network interfaces on Darwin.
func (n *neti) detectDarwinInterfaces() ([]Interface, error) {
	detectors := []func() ([]Interface, error){
		n.getMacIfconfigInterfaces,
		n.getMacSystemConfigInterfaces,
		// Detector via: `networksetup -listallhardwareports`
		// Detector via: `netstat -I <interface_name>`
		n.getMacGatewayDetails,
	}

	var errs []error
	interfaces := []Interface{}
	for _, detectFn := range detectors {
		detectedInterfaces, err := detectFn()
		if err != nil {
			log.Debug().Err(err).Msg("os.network.interface> unable to detect network interfaces")
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

func (n *neti) getMacSystemConfigInterfaces() ([]Interface, error) {
	var interfaces []Interface
	content, err := afero.ReadFile(
		n.connection.FileSystem(),
		"/Library/Preferences/SystemConfiguration/NetworkInterfaces.plist",
	)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if _, err := plist.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	if entries, ok := result["Interfaces"].([]any); ok {
		for _, entry := range entries {
			if iface, ok := entry.(map[string]any); ok {
				iName, ok := iface["BSD Name"].(string)
				if !ok {
					log.Trace().
						Interface("interface", iface).
						Str("detector", "SystemConfiguration").
						Msg("os.network.interface> unable to detect network interface")
					continue
				}

				// Check if the interface is hidden, if so, don't add it
				if hidden, ok := iface["HiddenInterface"].(bool); ok && hidden {
					log.Debug().
						Interface("interface", iface).
						Str("detector", "SystemConfiguration").
						Msg("os.network.interface> found hidden network interface, skipping")
					continue
				}

				intf := Interface{
					Name: iName,
				}

				if active, ok := iface["Active"].(bool); ok {
					intf.Active = &active
				}

				interfaces = append(interfaces, intf)
			}
		}
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "SystemConfiguration").
		Msg("os.network.interfaces> discovered")
	return interfaces, nil
}

func (n *neti) getMacGatewayDetails() (interfaces []Interface, err error) {
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		// we are looking for lines like this one
		//
		// Destination        Gateway            Flags               Netif
		// default            192.168.86.1       UGScg                en0
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) > 3 && isDefaultRoute(fields[0]) {

			gatewayVersion := IPv4
			if strings.Contains(fields[1], ":") {
				gatewayVersion = IPv6
			}

			interfaces = append(interfaces, Interface{
				Name: fields[3],
				enrichments: func(in *Interface) {
					for i := range in.IPAddresses {
						version, ok := in.IPAddresses[i].Version()
						if !ok {
							continue
						}
						if version == gatewayVersion {
							in.IPAddresses[i].Gateway = fields[1]
						}
					}
				},
			})
		}
	}
	return
}

func (n *neti) getMacIfconfigInterfaces() (interfaces []Interface, err error) {
	output, err := n.RunCommand("ifconfig")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentInterface *Interface
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Match interface name
		if strings.Contains(line, "flags=") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				if currentInterface != nil {
					interfaces = append(interfaces, *currentInterface)
				}
				currentInterface = &Interface{Name: strings.TrimSuffix(fields[0], ":")}
				if strings.HasPrefix(currentInterface.Name, "vmnet") {
					currentInterface.Virtual = convert.ToPtr(true)
				}
			}
		}

		if currentInterface != nil {
			// Match MAC address
			if strings.Contains(line, "ether") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					currentInterface.SetMAC(fields[1])
				}
			}

			// Match IPv4 address, CIDR, and netmask
			if strings.HasPrefix(line, "inet ") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					ip := fields[1]
					ipv4, ok := NewIPAddress(ip)
					if !ok {
						log.Trace().Str("ip", ip).Msg("not a valid ipaddress, skipping")
						continue
					}
					if len(fields) > 3 {
						// netmask found
						netmask := fields[3]
						ipv4 = NewIPv4WithMask(ip, netmask)
					}
					currentInterface.AddOrUpdateIP(ipv4)
				}
			}

			// Match IPv6 address, CIDR, and netmask
			if strings.HasPrefix(line, "inet6 ") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					ip := fields[1]
					// Check if the ip address has a scope id like "fe80::1%lo0"
					if strings.Contains(ip, "%") {
						ipWithScope := strings.Split(ip, "%")
						ip = ipWithScope[0]
						if ipWithScope[1] != currentInterface.Name {
							log.Debug().
								Str("scope_id", ipWithScope[1]).
								Str("interface_name", currentInterface.Name).
								Msg("ipv6 scope id and interface name mismatched")
						}
					}
					ipv6, ok := NewIPAddress(ip)
					if !ok {
						log.Trace().Str("ip", ip).Msg("not a valid ipaddress, skipping")
						continue
					}
					if len(fields) > 3 {
						// prefix length found
						prefixLength := parseInt(fields[3])
						ipv6 = NewIPv6WithPrefixLength(ip, prefixLength)
					}
					currentInterface.AddOrUpdateIP(ipv6)
				}
			}

			// Match MTU
			if strings.Contains(line, "mtu") {
				fields := strings.Fields(line)
				for i, f := range fields {
					if f == "mtu" && i+1 < len(fields) {
						currentInterface.MTU = parseInt(fields[i+1])
					}
				}
			}

			// Match status [active/inactive]
			if strings.Contains(line, "status:") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					if fields[1] == "active" {
						currentInterface.Active = convert.ToPtr(true)
					} else if fields[1] == "inactive" {
						currentInterface.Active = convert.ToPtr(false)
					}
				}
			}

			// Match flags
			if strings.Contains(line, "flags=") {
				flagsMatch := regexp.MustCompile(`flags=([0-9]+)<([^>]+)>`).FindStringSubmatch(line)
				if len(flagsMatch) > 2 {
					currentInterface.Flags = strings.Split(flagsMatch[2], ",")
				}
			}
		}
	}

	if currentInterface != nil {
		interfaces = append(interfaces, *currentInterface)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.ifconfig").
		Msg("os.network.interfaces> discovered")
	return interfaces, nil
}

func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}
