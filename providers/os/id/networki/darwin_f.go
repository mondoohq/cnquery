// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"strings"

	"github.com/cockroachdb/errors"
)

// detectDarwinRoutes detects network routes on macOS using netstat -nr
func (n *neti) detectDarwinRoutes() ([]Route, error) {
	output, err := n.RunCommand("netstat -nr")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get routes via netstat")
	}

	return n.parseNetstatOutput(output)
}

// parseNetstatOutput parses netstat -nr output
// Format:
// Routing tables
// Internet:
// Destination        Gateway            Flags        Netif Expire
// default            192.168.1.1        UGSc           en0
// 127                127.0.0.1          UCS            lo0
// 192.168.1          link#4             UCS            en0      !
// 192.168.1.1        0:1:2:3:4:5        UHLWIir        en0    3
// 192.168.1.255      ff:ff:ff:ff:ff:ff  UHLWbI         en0
func (n *neti) parseNetstatOutput(output string) ([]Route, error) {
	var routes []Route
	scanner := bufio.NewScanner(strings.NewReader(output))

	currentTable := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Detect table headers
		if strings.HasPrefix(line, "Internet:") {
			currentTable = "IPv4"
			continue
		}
		if strings.HasPrefix(line, "Internet6:") {
			currentTable = "IPv6"
			continue
		}

		// Skip header lines
		if strings.HasPrefix(line, "Destination") || strings.HasPrefix(line, "Routing tables") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		destination := fields[0]
		gateway := fields[1]
		flagsStr := fields[2]
		iface := fields[3]

		// Parse flags (convert flags string to int)
		// Flags: U (up), G (gateway), H (host), D (dynamic), M (modified), etc.
		flags := int64(0)
		for _, flag := range flagsStr {
			switch flag {
			case 'U':
				flags |= 0x1 // Up
			case 'G':
				flags |= 0x2 // Gateway
			case 'H':
				flags |= 0x4 // Host
			case 'D':
				flags |= 0x8 // Dynamic
			case 'M':
				flags |= 0x10 // Modified
			}
		}

		// Normalize destination
		if destination == "default" {
			if currentTable == "IPv6" {
				destination = "::/0"
			} else {
				destination = "0.0.0.0/0"
			}
		}

		// link-local or broadcast routes
		if gateway == "link#4" || strings.HasPrefix(gateway, "ff:ff:ff") {
			continue
		}

		// For IPv6, gateway might be in different format
		if currentTable == "IPv6" && !strings.Contains(gateway, ":") && gateway != "*" {
			// Skip invalid IPv6 gateways
			if gateway != "*" {
				continue
			}
		}

		routes = append(routes, Route{
			Destination: destination,
			Gateway:     gateway,
			Flags:       flags,
			Interface:   iface,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error parsing netstat output")
	}

	return routes, nil
}
