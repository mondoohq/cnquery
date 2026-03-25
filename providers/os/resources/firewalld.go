// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

type mqlFirewalldInternal struct {
	fetched      bool
	cacheStatus  string
	cacheDefault string
	lock         sync.Mutex
}

func (f *mqlFirewalld) id() (string, error) {
	return "firewalld", nil
}

func (z *mqlFirewalldZone) id() (string, error) {
	return "firewalld/zone/" + z.Name.Data, nil
}

func (r *mqlFirewalldRichrule) id() (string, error) {
	return "firewalld/richrule/" + r.Rule.Data, nil
}

// fetchStatus lazily loads the firewalld status and default zone.
func (f *mqlFirewalld) fetchStatus() error {
	if f.fetched {
		return nil
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.fetched {
		return nil
	}

	conn, ok := f.MqlRuntime.Connection.(shared.Connection)
	if !ok || !conn.Capabilities().Has(shared.Capability_RunCommand) {
		f.cacheStatus = "not installed"
		f.fetched = true
		return nil
	}

	// Check if firewalld is running
	o, err := CreateResource(f.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("firewall-cmd --state"),
	})
	if err != nil {
		f.cacheStatus = "not installed"
		f.fetched = true
		return nil
	}
	cmd := o.(*mqlCommand)
	state := strings.TrimSpace(cmd.Stdout.Data)
	if cmd.GetExitcode().Data != 0 || state != "running" {
		f.cacheStatus = "not running"
		f.fetched = true
		return nil
	}
	f.cacheStatus = "running"

	// Get default zone
	o, err = CreateResource(f.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("firewall-cmd --get-default-zone"),
	})
	if err != nil {
		return err
	}
	cmd = o.(*mqlCommand)
	if cmd.GetExitcode().Data != 0 {
		return fmt.Errorf("firewall-cmd --get-default-zone failed: %s", cmd.Stderr.Data)
	}
	f.cacheDefault = strings.TrimSpace(cmd.Stdout.Data)

	f.fetched = true
	return nil
}

func (f *mqlFirewalld) status() (string, error) {
	if err := f.fetchStatus(); err != nil {
		return "", err
	}
	return f.cacheStatus, nil
}

func (f *mqlFirewalld) defaultZone() (string, error) {
	if err := f.fetchStatus(); err != nil {
		return "", err
	}
	return f.cacheDefault, nil
}

func (f *mqlFirewalld) zones() ([]any, error) {
	if err := f.fetchStatus(); err != nil {
		return nil, err
	}
	if f.cacheStatus != "running" {
		return nil, nil
	}

	o, err := CreateResource(f.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("firewall-cmd --list-all-zones"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if cmd.GetExitcode().Data != 0 {
		return nil, fmt.Errorf("firewall-cmd --list-all-zones failed: %s", cmd.Stderr.Data)
	}

	zones := parseFirewalldZones(cmd.Stdout.Data)

	var res []any
	for _, z := range zones {
		var richRules []any
		for _, rr := range z.richRules {
			parsed := parseFirewalldRichRule(rr)
			r, err := CreateResource(f.MqlRuntime, "firewalld.richrule", map[string]*llx.RawData{
				"family":      llx.StringData(parsed.family),
				"rule":        llx.StringData(rr),
				"source":      llx.StringData(parsed.source),
				"destination": llx.StringData(parsed.destination),
				"action":      llx.StringData(parsed.action),
			})
			if err != nil {
				return nil, err
			}
			richRules = append(richRules, r)
		}

		zone, err := CreateResource(f.MqlRuntime, "firewalld.zone", map[string]*llx.RawData{
			"name":               llx.StringData(z.name),
			"target":             llx.StringData(z.target),
			"icmpBlockInversion": llx.BoolData(z.icmpBlockInversion),
			"active":             llx.BoolData(z.active),
			"interfaces":         llx.ArrayData(stringsToAny(z.interfaces), types.String),
			"sources":            llx.ArrayData(stringsToAny(z.sources), types.String),
			"services":           llx.ArrayData(stringsToAny(z.services), types.String),
			"ports":              llx.ArrayData(stringsToAny(z.ports), types.String),
			"masquerade":         llx.BoolData(z.masquerade),
			"forwardPorts":       llx.ArrayData(stringsToAny(z.forwardPorts), types.String),
			"sourcePorts":        llx.ArrayData(stringsToAny(z.sourcePorts), types.String),
			"icmpBlocks":         llx.ArrayData(stringsToAny(z.icmpBlocks), types.String),
			"richRules":          llx.ArrayData(richRules, types.Resource("firewalld.richrule")),
			"protocols":          llx.ArrayData(stringsToAny(z.protocols), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, zone)
	}
	return res, nil
}

// stringsToAny converts a string slice to []any for llx.ArrayData.
func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// parsedZone holds the parsed output of a single zone block from firewall-cmd --list-all-zones.
type parsedZone struct {
	name               string
	target             string
	icmpBlockInversion bool
	active             bool
	interfaces         []string
	sources            []string
	services           []string
	ports              []string
	masquerade         bool
	forwardPorts       []string
	sourcePorts        []string
	icmpBlocks         []string
	richRules          []string
	protocols          []string
}

// parseFirewalldZones parses the output of `firewall-cmd --list-all-zones`.
//
// Example format:
//
//	public (active)
//	  target: default
//	  icmp-block-inversion: no
//	  interfaces: eth0
//	  sources:
//	  services: cockpit dhcpv6-client ssh
//	  ports: 8080/tcp
//	  protocols:
//	  forward: yes
//	  masquerade: no
//	  forward-ports:
//	  source-ports:
//	  icmp-blocks:
//	  rich rules:
//	    rule family="ipv4" source address="10.0.0.0/8" accept
func parseFirewalldZones(output string) []parsedZone {
	var zones []parsedZone
	var current *parsedZone
	inRichRules := false

	for line := range strings.SplitSeq(output, "\n") {
		// Zone header: starts at column 0, non-empty, not indented
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			if current != nil {
				zones = append(zones, *current)
			}
			z := parsedZone{}
			header := strings.TrimSpace(line)
			// Parse "public (active)" or "dmz"
			if idx := strings.Index(header, " ("); idx != -1 {
				z.name = header[:idx]
				z.active = strings.Contains(header[idx:], "active")
			} else {
				z.name = header
			}
			current = &z
			inRichRules = false
			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Rich rules continuation: indented lines after "rich rules:" header
		if inRichRules {
			if strings.HasPrefix(trimmed, "rule ") {
				current.richRules = append(current.richRules, trimmed)
				continue
			}
			// If it doesn't start with "rule ", it's a new key
			inRichRules = false
		}

		// key: value pairs
		key, value, found := strings.Cut(trimmed, ": ")
		if !found {
			// Handle "rich rules:" with no value on same line
			if strings.TrimSuffix(trimmed, ":") == "rich rules" {
				inRichRules = true
				continue
			}
			// Also handle lines like "key:" with no value
			continue
		}
		value = strings.TrimSpace(value)

		switch key {
		case "target":
			current.target = value
		case "icmp-block-inversion":
			current.icmpBlockInversion = value == "yes"
		case "interfaces":
			current.interfaces = splitNonEmpty(value)
		case "sources":
			current.sources = splitNonEmpty(value)
		case "services":
			current.services = splitNonEmpty(value)
		case "ports":
			current.ports = splitNonEmpty(value)
		case "protocols":
			current.protocols = splitNonEmpty(value)
		case "masquerade":
			current.masquerade = value == "yes"
		case "forward-ports":
			current.forwardPorts = splitNonEmpty(value)
		case "source-ports":
			current.sourcePorts = splitNonEmpty(value)
		case "icmp-blocks":
			current.icmpBlocks = splitNonEmpty(value)
		case "rich rules":
			inRichRules = true
			// Value on the same line as "rich rules:" is unusual but handle it
			if value != "" && strings.HasPrefix(value, "rule ") {
				current.richRules = append(current.richRules, value)
			}
		}
	}
	if current != nil {
		zones = append(zones, *current)
	}
	return zones
}

// splitNonEmpty splits a space-separated string and returns nil for empty input.
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

// parsedRichRule holds the parsed components of a firewalld rich rule.
type parsedRichRule struct {
	family      string
	source      string
	destination string
	action      string
}

// parseFirewalldRichRule extracts structured fields from a rich rule string.
// Example: `rule family="ipv4" source address="10.0.0.0/8" accept`
func parseFirewalldRichRule(rule string) parsedRichRule {
	var rr parsedRichRule

	// Extract family
	if _, after, ok := strings.Cut(rule, `family="`); ok {
		if val, _, ok := strings.Cut(after, `"`); ok {
			rr.family = val
		}
	}

	// Extract source address
	if _, after, ok := strings.Cut(rule, `source address="`); ok {
		if val, _, ok := strings.Cut(after, `"`); ok {
			rr.source = val
		}
	}

	// Extract destination address
	if _, after, ok := strings.Cut(rule, `destination address="`); ok {
		if val, _, ok := strings.Cut(after, `"`); ok {
			rr.destination = val
		}
	}

	// Extract action — the terminal action keyword is always the last token
	// in a rich rule string (e.g., "rule family=... source address=... accept")
	tokens := strings.Fields(rule)
	if len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		switch last {
		case "accept", "reject", "drop", "mark", "log":
			rr.action = last
		}
	}

	return rr
}
