// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/types"
)

func (f *mqlFirewalld) id() (string, error) {
	return "firewalld", nil
}

func (f *mqlFirewalld) running() (bool, error) {
	conn := f.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("firewall-cmd --state")
	if err != nil {
		return false, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return false, err
	}

	// firewall-cmd --state returns "running" when active, non-zero exit on failure
	state := strings.TrimSpace(string(data))
	return state == "running", nil
}

func (f *mqlFirewalld) defaultZone() (string, error) {
	conn := f.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("firewall-cmd --get-default-zone")
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	if cmd.ExitStatus != 0 {
		return "", nil
	}

	return strings.TrimSpace(string(data)), nil
}

func (f *mqlFirewalld) zones() ([]any, error) {
	conn := f.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("firewall-cmd --list-all-zones")
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		return nil, nil
	}

	zones := parseFirewalldZones(string(data))

	result := make([]any, 0, len(zones))
	for _, zone := range zones {
		zoneResource, err := CreateResource(f.MqlRuntime, "firewalld.zone", map[string]*llx.RawData{
			"name":               llx.StringData(zone.Name),
			"target":             llx.StringData(zone.Target),
			"icmpBlockInversion": llx.BoolData(zone.IcmpBlockInversion),
			"interfaces":         llx.ArrayData(toAnySlice(zone.Interfaces), types.String),
			"sources":            llx.ArrayData(toAnySlice(zone.Sources), types.String),
			"services":           llx.ArrayData(toAnySlice(zone.Services), types.String),
			"ports":              llx.ArrayData(toAnySlice(zone.Ports), types.String),
			"protocols":          llx.ArrayData(toAnySlice(zone.Protocols), types.String),
			"masquerade":         llx.BoolData(zone.Masquerade),
			"forwardPorts":       llx.ArrayData(toAnySlice(zone.ForwardPorts), types.String),
			"sourcePorts":        llx.ArrayData(toAnySlice(zone.SourcePorts), types.String),
			"icmpBlocks":         llx.ArrayData(toAnySlice(zone.IcmpBlocks), types.String),
			"richRules":          llx.ArrayData(toAnySlice(zone.RichRules), types.String),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, zoneResource)
	}

	return result, nil
}

func (z *mqlFirewalldZone) id() (string, error) {
	return "firewalld.zone/" + z.Name.Data, nil
}

// FirewalldZone represents a parsed firewalld zone
type FirewalldZone struct {
	Name               string
	Target             string
	IcmpBlockInversion bool
	Interfaces         []string
	Sources            []string
	Services           []string
	Ports              []string
	Protocols          []string
	Masquerade         bool
	ForwardPorts       []string
	SourcePorts        []string
	IcmpBlocks         []string
	RichRules          []string
}

// parseFirewalldZones parses the output of `firewall-cmd --list-all-zones`
func parseFirewalldZones(output string) []FirewalldZone {
	var zones []FirewalldZone
	var currentZone *FirewalldZone

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Empty line may signal end of zone, but we handle new zone start
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Zone header line starts without whitespace and ends with zone name
		// Format: "zonename" or "zonename (active)"
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Save previous zone if exists
			if currentZone != nil {
				zones = append(zones, *currentZone)
			}
			// Start new zone
			zoneName := strings.TrimSpace(line)
			// Remove (active) suffix if present
			zoneName = strings.TrimSuffix(zoneName, " (active)")
			currentZone = &FirewalldZone{Name: zoneName}
			continue
		}

		// Parse zone properties (indented lines)
		if currentZone != nil {
			line = strings.TrimSpace(line)
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "target":
				currentZone.Target = value
			case "icmp-block-inversion":
				currentZone.IcmpBlockInversion = value == "yes"
			case "interfaces":
				currentZone.Interfaces = parseSpaceSeparated(value)
			case "sources":
				currentZone.Sources = parseSpaceSeparated(value)
			case "services":
				currentZone.Services = parseSpaceSeparated(value)
			case "ports":
				currentZone.Ports = parseSpaceSeparated(value)
			case "protocols":
				currentZone.Protocols = parseSpaceSeparated(value)
			case "masquerade":
				currentZone.Masquerade = value == "yes"
			case "forward-ports":
				currentZone.ForwardPorts = parseFirewalldForwardPorts(value)
			case "source-ports":
				currentZone.SourcePorts = parseSpaceSeparated(value)
			case "icmp-blocks":
				currentZone.IcmpBlocks = parseSpaceSeparated(value)
			case "rich rules":
				currentZone.RichRules = parseFirewalldRichRules(value)
			}
		}
	}

	// Don't forget the last zone
	if currentZone != nil {
		zones = append(zones, *currentZone)
	}

	return zones
}

// parseSpaceSeparated splits a space-separated string into a slice
func parseSpaceSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Fields(value)
}

// parseFirewalldForwardPorts parses forward-ports which may be on multiple lines
// or contain special formatting
func parseFirewalldForwardPorts(value string) []string {
	if value == "" {
		return []string{}
	}
	// Forward ports can be space-separated or newline-separated
	// Each entry looks like: port=80:proto=tcp:toport=8080:toaddr=
	return strings.Fields(value)
}

// parseFirewalldRichRules parses rich rules which may contain spaces
func parseFirewalldRichRules(value string) []string {
	if value == "" {
		return []string{}
	}
	// Rich rules are typically one per line with tab indentation for continuation
	// For single-line output, they may be space-separated but enclosed in quotes
	// Simple approach: treat each non-empty segment as a rule
	rules := []string{}
	// Check if the value contains multiple rules (usually newline-separated in real output)
	for _, rule := range strings.Split(value, "\n") {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			rules = append(rules, rule)
		}
	}
	// If no newlines, the whole value is one rule (if non-empty)
	if len(rules) == 0 && value != "" {
		rules = append(rules, value)
	}
	return rules
}

// toAnySlice converts a []string to []any for llx.ArrayData
func toAnySlice(strs []string) []any {
	result := make([]any, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
