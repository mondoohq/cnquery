// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// detectWindowsInterfaces detects network interfaces on Windows.
func (n *neti) detectWindowsInterfaces() ([]Interface, error) {
	detectors := []func() ([]Interface, error){
		n.getWindowsIpconfigCmdInterfaces,
		n.getWindowsGetNetIPInterfaceCmdInterfaces,
	}

	var errs []error
	interfaces := []Interface{}
	for _, detectFn := range detectors {
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

func (n *neti) getWindowsGetNetIPInterfaceCmdInterfaces() (interfaces []Interface, err error) {
	cmd := `
  Get-NetIPInterface | 
    Select-Object InterfaceIndex, InterfaceAlias, NlMtu, ConnectionState, AddressFamily,
    @{ Name='MacAddress'; Expression={ (Get-NetAdapter -InterfaceIndex $_.InterfaceIndex).MacAddress } },
    @{ Name='IPAddresses'; Expression={
      (Get-NetIPAddress -InterfaceIndex $_.InterfaceIndex) |
      Select-Object InterfaceAlias, AddressFamily, IPAddress, PrefixLength |
      ConvertTo-Json
    } },
    @{ Name='Virtual'; Expression={ (Get-NetAdapter -InterfaceIndex $_.InterfaceIndex).Virtual } } |
    ConvertTo-Json
	`
	output, err := n.RunCommand(cmd)
	if err != nil {
		return nil, err
	}

	var netInterfaces []map[string]any
	err = json.Unmarshal([]byte(output), &netInterfaces)
	if err != nil {
		return nil, err
	}

	log.Trace().Interface("output", netInterfaces).Msg("os.network.interface> net interface cmd")

	for _, adapter := range netInterfaces {
		iinterface := Interface{
			Name: adapter["InterfaceAlias"].(string),
		}

		// Get MAC address
		if value, ok := adapter["MacAddress"].(string); ok {
			iinterface.SetMAC(value)
		}
		// Get MTU
		if value, ok := adapter["NlMtu"].(float64); ok {
			iinterface.MTU = int(value)
		}

		// Get Status
		if state, ok := adapter["ConnectionState"].(float64); ok {
			active := true
			if state == 0 {
				active = false
			}
			iinterface.Active = &active
		}

		// Detect virtual interface
		if virtual, ok := adapter["Virtual"].(bool); ok {
			iinterface.Virtual = &virtual
		}

		// Get IP Addresses (v4 or v6) in JSON format
		if data, ok := adapter["IPAddresses"].(string); ok {
			var ipaddresses []map[string]any
			err = json.Unmarshal([]byte(data), &ipaddresses)
			if err != nil {
				log.Debug().Err(err).
					Str("data", data).
					Str("detector", "cmd.Get-NetIPInterface").
					Msg("os.network.interface> unable to detect IPAddresses")
			}

			var (
				ipaddress IPAddress
				valid     bool
			)
			for _, ipMap := range ipaddresses {
				if ip, ok := ipMap["IPAddress"].(string); ok {
					// Get the prefix length
					if prefixLength, ok := ipMap["PrefixLength"].(float64); ok {
						ipaddress, valid = NewIPWithPrefixLength(ip, int(prefixLength))
					} else {
						// No prefix, plain ip address
						ipaddress, valid = NewIPAddress(ip)
					}

					if valid {
						iinterface.AddOrUpdateIP(ipaddress)
					}
				}
			}
		}

		interfaces = append(interfaces, iinterface)
	}

	return
}

func (n *neti) getWindowsIpconfigCmdInterfaces() (interfaces []Interface, err error) {
	output, err := n.RunCommand("ipconfig /all")
	if err != nil {
		return nil, err
	}

	var (
		ips                  []IPAddress
		gateways             []string
		currentInterface     *Interface
		interfaceHeaderRegex = regexp.MustCompile(`^(Ethernet|Wireless|Bluetooth|VPN|Local Area Connection|Wi-Fi|Cellular|Tunnel) adapter (.+):$`)
	)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// We are looking for an output like this one to identify a new interface
		//
		// Ethernet adapter Ethernet0:
		if matches := interfaceHeaderRegex.FindStringSubmatch(line); matches != nil {
			if currentInterface != nil {
				updateWindowsNetInterface(currentInterface, ips, gateways)
				interfaces = append(interfaces, *currentInterface)
			}

			// New interface initialization
			currentInterface = &Interface{Name: matches[2]}
			ips = make([]IPAddress, 0)
			gateways = make([]string, 0)
		}

		if currentInterface != nil {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			switch {
			case strings.HasPrefix(line, "Physical Address"):
				currentInterface.SetMAC(lastField(fields))
			case strings.HasPrefix(line, "IPv4 Address"):
				fallthrough
			case strings.HasPrefix(line, "IPv6 Address"):
				ip, ok := NewIPAddress(cleanIPString(lastField(fields)))
				if ok {
					ips = append(ips, ip)
				}
			case strings.HasPrefix(line, "Subnet Mask"):
				// Subnet mask are only valid for IPv4
				subnet := lastField(fields)
				for i := range ips {
					if version, ok := ips[i].Version(); ok && version == IPv4 {
						ips[i] = NewIPv4WithMask(ips[i].IP.String(), subnet)
					}
				}
			case strings.HasPrefix(line, "Default Gateway"):
				// collect the gateways found to inject them as part of the enrichments
				gateways = append(gateways, lastField(fields))
			}
		}
	}

	if currentInterface != nil {
		updateWindowsNetInterface(currentInterface, ips, gateways)
		interfaces = append(interfaces, *currentInterface)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.ipconfig_/all").
		Msg("os.network.interfaces> discovered")
	return
}

func lastField(fields []string) string {
	if len(fields) > 2 {
		return fields[len(fields)-1]
	}
	return ""
}

func cleanIPString(ip string) string {
	re := regexp.MustCompile(`\(.*?\)$`)
	return strings.TrimSpace(re.ReplaceAllString(ip, ""))
}

func updateWindowsNetInterface(currentInterface *Interface, ips []IPAddress, gateways []string) {
	currentInterface.AddOrUpdateIP(ips...)
	if len(gateways) == 0 {
		// no enrichments needed
		return
	}
	currentInterface.enrichments = func(in *Interface) {
		for g := range gateways {
			// IPv4 (default)
			gateway := gateways[g]
			gatewayVersion := IPv4
			if strings.Contains(gateway, ":") {
				// IPv6
				gatewayVersion = IPv6
			}
			for i := range in.IPAddresses {
				if version, ok := in.IPAddresses[i].Version(); ok && version == gatewayVersion {
					in.IPAddresses[i].Gateway = gateway
				}
			}
		}
	}
}
