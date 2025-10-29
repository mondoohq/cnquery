// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/types"
)

func (f *mqlFirewalld) zones() ([]any, error) {
	conn := f.MqlRuntime.Connection.(shared.Connection)

	defaultZone, err := runFirewallCmd(conn, "--get-default-zone")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve firewalld default zone: %w", err)
	}
	defaultZone = strings.TrimSpace(defaultZone)
	f.DefaultZone = plugin.TValue[string]{Data: defaultZone, State: plugin.StateIsSet}

	activeZonesRaw, err := runFirewallCmd(conn, "--get-active-zones")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active firewalld zones: %w", err)
	}
	activeZones := parseActiveZones(activeZonesRaw)
	f.ActiveZones = plugin.TValue[[]any]{Data: stringsToAny(activeZones), State: plugin.StateIsSet}

	zoneListRaw, err := runFirewallCmd(conn, "--get-zones")
	if err != nil {
		return nil, fmt.Errorf("failed to list firewalld zones: %w", err)
	}
	zoneNames := strings.Fields(zoneListRaw)
	activeLookup := make(map[string]struct{}, len(activeZones))
	for _, zone := range activeZones {
		activeLookup[zone] = struct{}{}
	}

	zones := make([]any, 0, len(zoneNames))
	for _, zoneName := range zoneNames {
		zoneName = strings.TrimSpace(zoneName)
		if zoneName == "" {
			continue
		}

		listAllRaw, err := runFirewallCmd(conn, "--zone="+zoneName, "--list-all")
		if err != nil {
			return nil, fmt.Errorf("failed to inspect firewalld zone %q: %w", zoneName, err)
		}

		zoneInfo := parseFirewalldZone(zoneName, listAllRaw)
		if _, ok := activeLookup[zoneName]; ok {
			zoneInfo.Active = true
		}

		richRulesRaw, err := runFirewallCmd(conn, "--zone="+zoneName, "--list-rich-rules")
		if err != nil {
			return nil, fmt.Errorf("failed to list rich rules for firewalld zone %q: %w", zoneName, err)
		}

		richRuleLines := parseRichRuleLines(richRulesRaw)
		parsedRules := make([]parsedFirewalldRule, 0, len(richRuleLines))
		for _, line := range richRuleLines {
			parsedRules = append(parsedRules, parseRichRule(line))
		}

		mqlZone, err := createFirewalldZoneResource(f.MqlRuntime, zoneInfo, parsedRules)
		if err != nil {
			return nil, err
		}
		zones = append(zones, mqlZone)
	}

	f.Zones = plugin.TValue[[]any]{Data: zones, State: plugin.StateIsSet}

	return zones, nil
}

func runFirewallCmd(conn shared.Connection, args ...string) (string, error) {
	cmdLine := "firewall-cmd"
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}

	cmd, err := conn.RunCommand(cmdLine)
	if err != nil {
		return "", err
	}

	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(stdout))
		}
		if msg == "" {
			msg = fmt.Sprintf("firewall-cmd %s failed with exit status %d", strings.Join(args, " "), cmd.ExitStatus)
		} else {
			msg = fmt.Sprintf("%s: %s", cmdLine, msg)
		}
		return "", errors.New(msg)
	}

	return string(stdout), nil
}

func parseActiveZones(raw string) []string {
	lines := strings.Split(raw, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		res = append(res, strings.TrimSpace(line))
	}
	return res
}

type parsedFirewalldZone struct {
	Name               string
	Target             string
	Active             bool
	Interfaces         []string
	Sources            []string
	Services           []string
	Ports              []string
	Protocols          []string
	Masquerade         bool
	ForwardPorts       []string
	SourcePorts        []string
	IcmpBlocks         []string
	IcmpBlockInversion bool
	Raw                string
}

func parseFirewalldZone(name, raw string) parsedFirewalldZone {
	lines := strings.Split(raw, "\n")
	zone := parsedFirewalldZone{
		Name: name,
		Raw:  strings.TrimSpace(raw),
	}

	if len(lines) > 0 && strings.Contains(lines[0], "(active)") {
		zone.Active = true
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "target":
			zone.Target = value
		case "interfaces":
			zone.Interfaces = splitList(value)
		case "sources":
			zone.Sources = splitList(value)
		case "services":
			zone.Services = splitList(value)
		case "ports":
			zone.Ports = splitList(value)
		case "protocols":
			zone.Protocols = splitList(value)
		case "masquerade":
			zone.Masquerade = parseBool(value)
		case "forward-ports":
			zone.ForwardPorts = splitList(value)
		case "source-ports":
			zone.SourcePorts = splitList(value)
		case "icmp-blocks":
			zone.IcmpBlocks = splitList(value)
		case "icmp-block-inversion":
			zone.IcmpBlockInversion = parseBool(value)
		}
	}

	return zone
}

func splitList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	fields := strings.Fields(value)
	res := make([]string, 0, len(fields))
	for _, v := range fields {
		if v != "" {
			res = append(res, v)
		}
	}
	return res
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "true", "on", "1":
		return true
	default:
		return false
	}
}

func parseRichRuleLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		res = append(res, line)
	}
	return res
}

type parsedRuleEndpoint struct {
	Address  string
	Ipset    string
	Mac      string
	HasValue bool
	HasNot   bool
	Not      parsedRuleQualifier
}

type parsedRuleQualifier struct {
	Address string
	Ipset   string
	Mac     string
}

type parsedFirewalldRule struct {
	Raw       string
	Family    string
	Priority  *int
	Source    parsedRuleEndpoint
	Dest      parsedRuleEndpoint
	Service   string
	Port      string
	LogPrefix string
	LogLevel  string
	Action    string
}

func parseRichRule(line string) parsedFirewalldRule {
	rule := parsedFirewalldRule{
		Raw: strings.TrimSpace(line),
	}

	tokens := tokenizeRichRule(line)
	section := ""
	inNot := false
	var action string
	var portValue string
	var portProtocol string

	for _, token := range tokens {
		if token == "" {
			continue
		}

		switch token {
		case "rule":
			continue
		case "source":
			section = "source"
			inNot = false
			continue
		case "destination":
			section = "destination"
			inNot = false
			continue
		case "service":
			section = "service"
			continue
		case "port":
			section = "port"
			portValue = ""
			portProtocol = ""
			continue
		case "log":
			section = "log"
			continue
		case "limit":
			continue
		case "not":
			if section == "source" {
				inNot = true
				rule.Source.HasNot = true
			} else if section == "destination" {
				inNot = true
				rule.Dest.HasNot = true
			}
			continue
		}

		if strings.Contains(token, "=") {
			parts := strings.SplitN(token, "=", 2)
			key := parts[0]
			value := parts[1]

			switch section {
			case "source":
				assignEndpointValue(&rule.Source, key, value, inNot)
			case "destination":
				assignEndpointValue(&rule.Dest, key, value, inNot)
			case "service":
				if key == "name" {
					rule.Service = value
				}
			case "port":
				if key == "port" {
					portValue = value
				}
				if key == "protocol" {
					portProtocol = value
				}
			case "log":
				if key == "prefix" {
					rule.LogPrefix = value
				}
				if key == "level" {
					rule.LogLevel = value
				}
			default:
				switch key {
				case "family":
					rule.Family = value
				case "priority":
					if v, err := strconv.Atoi(value); err == nil {
						rule.Priority = &v
					}
				case "action":
					rule.Action = value
				}
			}
			continue
		}

		if token != "source" && token != "destination" && token != "service" && token != "port" && token != "log" && token != "not" {
			action = token
		}
	}

	if rule.Port == "" && portValue != "" {
		if portProtocol != "" {
			rule.Port = portValue + "/" + portProtocol
		} else {
			rule.Port = portValue
		}
	}

	if rule.Action == "" {
		rule.Action = action
	}

	return rule
}

func assignEndpointValue(ep *parsedRuleEndpoint, key, value string, inNot bool) {
	switch key {
	case "address":
		if inNot {
			ep.Not.Address = value
			ep.HasValue = true
			return
		}
		ep.Address = value
		ep.HasValue = true
	case "ipset":
		if inNot {
			ep.Not.Ipset = value
			ep.HasValue = true
			return
		}
		ep.Ipset = value
		ep.HasValue = true
	case "mac":
		if inNot {
			ep.Not.Mac = value
			ep.HasValue = true
			return
		}
		ep.Mac = value
		ep.HasValue = true
	}
}

func tokenizeRichRule(raw string) []string {
	raw = strings.TrimSpace(raw)
	tokens := []string{}
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch ch {
		case '"':
			inQuote = !inQuote
			continue
		case '\\':
			if inQuote && i+1 < len(raw) {
				i++
				current.WriteByte(raw[i])
			} else {
				current.WriteByte(ch)
			}
			continue
		case ' ', '\t':
			if inQuote {
				current.WriteByte(ch)
				continue
			}
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func createFirewalldZoneResource(runtime *plugin.Runtime, zone parsedFirewalldZone, rules []parsedFirewalldRule) (*mqlFirewalldZone, error) {
	ruleResources := make([]any, 0, len(rules))
	for idx, rule := range rules {
		ruleRes, err := createFirewalldRuleResource(runtime, zone.Name, idx, rule)
		if err != nil {
			return nil, err
		}
		ruleResources = append(ruleResources, ruleRes)
	}

	args := map[string]*llx.RawData{
		"__id":               llx.StringData(zone.Name),
		"name":               stringOrNil(zone.Name),
		"target":             stringOrNil(zone.Target),
		"active":             llx.BoolData(zone.Active),
		"interfaces":         stringArrayData(zone.Interfaces),
		"sources":            stringArrayData(zone.Sources),
		"services":           stringArrayData(zone.Services),
		"ports":              stringArrayData(zone.Ports),
		"protocols":          stringArrayData(zone.Protocols),
		"masquerade":         llx.BoolData(zone.Masquerade),
		"forwardPorts":       stringArrayData(zone.ForwardPorts),
		"sourcePorts":        stringArrayData(zone.SourcePorts),
		"icmpBlocks":         stringArrayData(zone.IcmpBlocks),
		"icmpBlockInversion": llx.BoolData(zone.IcmpBlockInversion),
		"richRules":          llx.ArrayData(ruleResources, types.Resource("firewalld.rule")),
		"raw":                stringOrNil(zone.Raw),
	}

	zoneRes, err := CreateResource(runtime, "firewalld.zone", args)
	if err != nil {
		return nil, err
	}

	return zoneRes.(*mqlFirewalldZone), nil
}

func createFirewalldRuleResource(runtime *plugin.Runtime, zoneName string, idx int, rule parsedFirewalldRule) (*mqlFirewalldRule, error) {
	ruleID := fmt.Sprintf("%s/rule/%d", zoneName, idx)

	var sourceRes plugin.Resource
	var err error
	if rule.Source.HasValue || rule.Source.HasNot {
		sourceRes, err = createRuleEndpointResource(runtime, zoneName, ruleID, "source", rule.Source)
		if err != nil {
			return nil, err
		}
	}

	var destRes plugin.Resource
	if rule.Dest.HasValue || rule.Dest.HasNot {
		destRes, err = createRuleEndpointResource(runtime, zoneName, ruleID, "destination", rule.Dest)
		if err != nil {
			return nil, err
		}
	}

	args := map[string]*llx.RawData{
		"__id":   llx.StringData(ruleID),
		"raw":    stringOrNil(rule.Raw),
		"family": stringOrNil(rule.Family),
		"service": func() *llx.RawData {
			if rule.Service == "" {
				return llx.NilData
			}
			return llx.StringData(rule.Service)
		}(),
		"port": func() *llx.RawData {
			if rule.Port == "" {
				return llx.NilData
			}
			return llx.StringData(rule.Port)
		}(),
		"logPrefix": func() *llx.RawData {
			if rule.LogPrefix == "" {
				return llx.NilData
			}
			return llx.StringData(rule.LogPrefix)
		}(),
		"logLevel": func() *llx.RawData {
			if rule.LogLevel == "" {
				return llx.NilData
			}
			return llx.StringData(rule.LogLevel)
		}(),
		"action": func() *llx.RawData {
			if rule.Action == "" {
				return llx.NilData
			}
			return llx.StringData(rule.Action)
		}(),
	}

	if rule.Priority != nil {
		args["priority"] = llx.IntData(int64(*rule.Priority))
	} else {
		args["priority"] = llx.NilData
	}

	if sourceRes != nil {
		args["source"] = llx.ResourceData(sourceRes, "firewalld.ruleEndpoint")
	} else {
		args["source"] = llx.NilData
	}

	if destRes != nil {
		args["destination"] = llx.ResourceData(destRes, "firewalld.ruleEndpoint")
	} else {
		args["destination"] = llx.NilData
	}

	ruleRes, err := CreateResource(runtime, "firewalld.rule", args)
	if err != nil {
		return nil, err
	}

	return ruleRes.(*mqlFirewalldRule), nil
}

func createRuleEndpointResource(runtime *plugin.Runtime, zoneName, ruleID, label string, ep parsedRuleEndpoint) (plugin.Resource, error) {
	endpointID := fmt.Sprintf("%s/%s/%s", zoneName, ruleID, label)

	args := map[string]*llx.RawData{
		"__id":    llx.StringData(endpointID),
		"address": stringOrNil(ep.Address),
		"ipset":   stringOrNil(ep.Ipset),
		"mac":     stringOrNil(ep.Mac),
	}

	if ep.HasNot {
		qualifierArgs := map[string]*llx.RawData{
			"__id":    llx.StringData(endpointID + "/not"),
			"address": stringOrNil(ep.Not.Address),
			"ipset":   stringOrNil(ep.Not.Ipset),
			"mac":     stringOrNil(ep.Not.Mac),
		}

		qualifierRes, err := CreateResource(runtime, "firewalld.ruleEndpointQualifier", qualifierArgs)
		if err != nil {
			return nil, err
		}
		args["not"] = llx.ResourceData(qualifierRes, "firewalld.ruleEndpointQualifier")
	} else {
		args["not"] = llx.NilData
	}

	res, err := CreateResource(runtime, "firewalld.ruleEndpoint", args)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func stringOrNil(s string) *llx.RawData {
	if strings.TrimSpace(s) == "" {
		return llx.NilData
	}
	return llx.StringData(s)
}

func stringArrayData(values []string) *llx.RawData {
	arr := stringsToAny(values)
	return llx.ArrayData(arr, types.String)
}

func stringsToAny(values []string) []any {
	res := make([]any, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		res = append(res, strings.TrimSpace(v))
	}
	return res
}
