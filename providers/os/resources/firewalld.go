// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/firewalld"
	"go.mondoo.com/cnquery/v12/providers/os/resources/parsers"
	"go.mondoo.com/cnquery/v12/types"
)

const defaultFirewalldConfig = "/etc/firewalld/firewalld.conf"

var firewalldZoneSearchPaths = []string{
	"/run/firewalld/zones",
	"/etc/firewalld/zones",
	"/usr/lib/firewalld/zones",
	"/usr/share/firewalld/zones",
}

type mqlFirewalldInternal struct {
	lock      sync.Mutex
	loaded    bool
	loadError error
}

type firewalldData struct {
	DefaultZone string
	ActiveZones []string
	Zones       []firewalldZoneData
}

type firewalldZoneData struct {
	Zone  parsedFirewalldZone
	Rules []parsedFirewalldRule
}

type firewalldZoneFile struct {
	Name    string
	Path    string
	Content string
}

func (f *mqlFirewalld) zones() ([]any, error) {
	if err := f.ensureLoaded(); err != nil {
		return nil, err
	}
	return f.Zones.Data, f.Zones.Error
}

func (f *mqlFirewalld) ensureLoaded() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.loaded {
		return f.loadError
	}

	data, err := loadFirewalldFromConfig(f.MqlRuntime)
	if err != nil {
		data, err = loadFirewalldFromCommand(f.MqlRuntime)
	}

	if err != nil {
		f.DefaultZone = plugin.TValue[string]{State: plugin.StateIsSet, Error: err}
		f.ActiveZones = plugin.TValue[[]any]{State: plugin.StateIsSet, Error: err}
		f.Zones = plugin.TValue[[]any]{State: plugin.StateIsSet, Error: err}
		f.loadError = err
		f.loaded = true
		return err
	}

	zoneResources := make([]any, 0, len(data.Zones))
	for _, zoneData := range data.Zones {
		zoneRes, err := createFirewalldZoneResource(f.MqlRuntime, zoneData.Zone, zoneData.Rules)
		if err != nil {
			f.DefaultZone = plugin.TValue[string]{State: plugin.StateIsSet, Error: err}
			f.ActiveZones = plugin.TValue[[]any]{State: plugin.StateIsSet, Error: err}
			f.Zones = plugin.TValue[[]any]{State: plugin.StateIsSet, Error: err}
			f.loadError = err
			f.loaded = true
			return err
		}
		zoneResources = append(zoneResources, zoneRes)
	}

	f.DefaultZone = plugin.TValue[string]{Data: data.DefaultZone, State: plugin.StateIsSet}
	f.ActiveZones = plugin.TValue[[]any]{Data: convert.SliceAnyToInterface(data.ActiveZones), State: plugin.StateIsSet}
	f.Zones = plugin.TValue[[]any]{Data: zoneResources, State: plugin.StateIsSet}

	f.loadError = nil
	f.loaded = true
	return nil
}

func loadFirewalldFromConfig(runtime *plugin.Runtime) (*firewalldData, error) {
	defaultZone, err := readFirewalldDefaultZone(runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to read firewalld config: %w", err)
	}

	zoneFiles := make(map[string]firewalldZoneFile)
	for _, dir := range firewalldZoneSearchPaths {
		files, err := collectFirewalldZoneFiles(runtime, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to read firewalld zones from %s: %w", dir, err)
		}

		for _, file := range files {
			if _, exists := zoneFiles[file.Name]; !exists {
				zoneFiles[file.Name] = file
			}
		}
	}

	if len(zoneFiles) == 0 {
		return nil, errors.New("no firewalld zone definitions found")
	}

	zoneNames := make([]string, 0, len(zoneFiles))
	for name := range zoneFiles {
		zoneNames = append(zoneNames, name)
	}
	sort.Strings(zoneNames)

	zones := make([]firewalldZoneData, 0, len(zoneNames))
	activeLookup := map[string]struct{}{}
	indexByName := make(map[string]int, len(zoneNames))

	for _, name := range zoneNames {
		file := zoneFiles[name]
		zoneData, err := parseFirewalldZoneFile(name, file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse firewalld zone %q: %w", file.Path, err)
		}
		indexByName[name] = len(zones)
		if zoneData.Zone.Active {
			activeLookup[name] = struct{}{}
		}
		zones = append(zones, zoneData)
	}

	if defaultZone == "" {
		if _, ok := zoneFiles["public"]; ok {
			defaultZone = "public"
		} else if len(zoneNames) > 0 {
			defaultZone = zoneNames[0]
		}
	}

	if defaultZone != "" {
		if _, ok := activeLookup[defaultZone]; !ok && len(activeLookup) == 0 {
			if idx, found := indexByName[defaultZone]; found {
				z := zones[idx]
				z.Zone.Active = true
				zones[idx] = z
				activeLookup[defaultZone] = struct{}{}
			}
		}
	}

	activeZones := make([]string, 0, len(activeLookup))
	for name := range activeLookup {
		activeZones = append(activeZones, name)
	}
	sort.Strings(activeZones)

	return &firewalldData{
		DefaultZone: defaultZone,
		ActiveZones: activeZones,
		Zones:       zones,
	}, nil
}

func readFirewalldDefaultZone(runtime *plugin.Runtime) (string, error) {
	fileRes, err := CreateResource(runtime, ResourceFile, map[string]*llx.RawData{
		"path": llx.StringData(defaultFirewalldConfig),
	})
	if err != nil {
		return "", err
	}
	file := fileRes.(*mqlFile)

	exists := file.GetExists()
	if exists.Error != nil {
		return "", exists.Error
	}
	if !exists.Data {
		return "", nil
	}

	content := file.GetContent()
	if content.Error != nil {
		return "", content.Error
	}

	ini := parsers.ParseIni(content.Data, "=")
	if ini == nil || len(ini.Fields) == 0 {
		return "", nil
	}

	root, ok := ini.Fields[""].(map[string]any)
	if !ok {
		return "", nil
	}

	for key, value := range root {
		if strings.EqualFold(strings.TrimSpace(key), "defaultzone") {
			if str, ok := value.(string); ok {
				return strings.TrimSpace(str), nil
			}
		}
	}

	return "", nil
}

func collectFirewalldZoneFiles(runtime *plugin.Runtime, dir string) ([]firewalldZoneFile, error) {
	if fsProvider, ok := runtime.Connection.(interface{ FileSystem() afero.Fs }); ok {
		fs := fsProvider.FileSystem()
		exists, err := afero.DirExists(fs, dir)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, nil
		}
		return collectZoneFilesFromFS(fs, dir)
	}

	fileRes, err := CreateResource(runtime, ResourceFile, map[string]*llx.RawData{
		"path": llx.StringData(dir),
	})
	if err != nil {
		return nil, err
	}
	file := fileRes.(*mqlFile)

	exists := file.GetExists()
	if exists.Error != nil {
		return nil, exists.Error
	}
	if !exists.Data {
		return nil, nil
	}

	perm := file.GetPermissions()
	if perm.Error != nil {
		return nil, perm.Error
	}
	if !perm.Data.IsDirectory.Data {
		return nil, nil
	}

	entries, err := getSortedPathFiles(runtime, dir)
	if err != nil {
		return nil, err
	}

	files := make([]firewalldZoneFile, 0, len(entries))
	for _, entry := range entries {
		mqlFile := entry.(*mqlFile)
		base := mqlFile.GetBasename()
		if base.Error != nil {
			return nil, base.Error
		}
		if !strings.EqualFold(filepath.Ext(base.Data), ".xml") {
			continue
		}

		name := strings.TrimSuffix(base.Data, filepath.Ext(base.Data))
		if strings.TrimSpace(name) == "" {
			continue
		}

		content := mqlFile.GetContent()
		if content.Error != nil {
			return nil, content.Error
		}

		files = append(files, firewalldZoneFile{
			Name:    name,
			Path:    mqlFile.Path.Data,
			Content: content.Data,
		})
	}

	return files, nil
}

func collectZoneFilesFromFS(fs afero.Fs, dir string) ([]firewalldZoneFile, error) {
	infos, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, err
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})

	files := make([]firewalldZoneFile, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(info.Name()), ".xml") {
			continue
		}

		name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		if strings.TrimSpace(name) == "" {
			continue
		}

		path := filepath.Join(dir, info.Name())
		data, err := afero.ReadFile(fs, path)
		if err != nil {
			return nil, err
		}

		files = append(files, firewalldZoneFile{
			Name:    name,
			Path:    path,
			Content: string(data),
		})
	}

	return files, nil
}

func parseFirewalldZoneFile(name string, file firewalldZoneFile) (firewalldZoneData, error) {
	var zoneXML firewalld.Zone
	if err := xml.Unmarshal([]byte(file.Content), &zoneXML); err != nil {
		return firewalldZoneData{}, err
	}

	zone := parsedFirewalldZone{
		Name:               name,
		Target:             zoneXML.Target,
		Interfaces:         interfacesToStrings(zoneXML.Interfaces),
		Sources:            sourcesToStrings(zoneXML.Sources),
		Services:           namesToStrings(zoneXML.Services),
		Ports:              portsToStrings(zoneXML.Ports),
		Protocols:          protocolsToStrings(zoneXML.Protocols),
		Masquerade:         zoneXML.Masquerade != nil,
		ForwardPorts:       forwardPortsToStrings(zoneXML.ForwardPorts),
		SourcePorts:        sourcePortsToStrings(zoneXML.SourcePorts),
		IcmpBlocks:         namesToStrings(zoneXML.IcmpBlocks),
		IcmpBlockInversion: zoneXML.IcmpBlockInversion != nil,
		Raw:                strings.TrimSpace(file.Content),
	}
	if len(zone.Interfaces) > 0 || len(zone.Sources) > 0 {
		zone.Active = true
	}

	rules := make([]parsedFirewalldRule, 0, len(zoneXML.Rules))
	for _, rule := range zoneXML.Rules {
		tokens := rule.ToTokens()
		ruleStr := strings.Join(tokens, " ")
		if strings.TrimSpace(ruleStr) == "" {
			continue
		}
		rules = append(rules, parseRichRule(ruleStr))
	}

	return firewalldZoneData{
		Zone:  zone,
		Rules: rules,
	}, nil
}

func interfacesToStrings(items []firewalld.Interface) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(item.Name)
		if val != "" {
			res = append(res, val)
		}
	}
	return res
}

func sourcesToStrings(items []firewalld.Source) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(item.Address)
		if val != "" {
			res = append(res, val)
		}
	}
	return res
}

func namesToStrings(items []firewalld.Name) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(item.Name)
		if val != "" {
			res = append(res, val)
		}
	}
	return res
}

func portsToStrings(items []firewalld.Port) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		port := strings.TrimSpace(item.Port)
		proto := strings.TrimSpace(item.Protocol)
		switch {
		case port == "" && proto == "":
			continue
		case port != "" && proto != "":
			res = append(res, port+"/"+proto)
		default:
			res = append(res, port+proto)
		}
	}
	return res
}

func sourcePortsToStrings(items []firewalld.Port) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		port := strings.TrimSpace(item.Port)
		proto := strings.TrimSpace(item.Protocol)
		if port == "" && proto == "" {
			continue
		}

		entry := []string{}
		if port != "" {
			entry = append(entry, "port="+port)
		}
		if proto != "" {
			entry = append(entry, "proto="+proto)
		}
		if len(entry) > 0 {
			res = append(res, strings.Join(entry, ":"))
		}
	}
	return res
}

func forwardPortsToStrings(items []firewalld.ForwardPort) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		parts := []string{}
		if v := strings.TrimSpace(item.Port); v != "" {
			parts = append(parts, "port="+v)
		}
		if v := strings.TrimSpace(item.Protocol); v != "" {
			parts = append(parts, "proto="+v)
		}
		if v := strings.TrimSpace(item.ToPort); v != "" {
			parts = append(parts, "toport="+v)
		}
		if v := strings.TrimSpace(item.ToAddr); v != "" {
			parts = append(parts, "toaddr="+v)
		}
		if len(parts) > 0 {
			res = append(res, strings.Join(parts, ":"))
		}
	}
	return res
}

func protocolsToStrings(items []firewalld.Protocol) []string {
	if len(items) == 0 {
		return nil
	}
	res := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(item.Value)
		if val != "" {
			res = append(res, val)
		}
	}
	return res
}

func loadFirewalldFromCommand(runtime *plugin.Runtime) (*firewalldData, error) {
	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return nil, errors.New("firewalld resource requires a shared OS connection")
	}

	defaultZoneRaw, err := runFirewallCmd(conn, "--get-default-zone")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve firewalld default zone: %w", err)
	}
	defaultZone := strings.TrimSpace(defaultZoneRaw)

	activeZonesRaw, err := runFirewallCmd(conn, "--get-active-zones")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active firewalld zones: %w", err)
	}
	activeZones := parseActiveZones(activeZonesRaw)
	activeLookup := make(map[string]struct{}, len(activeZones))
	for _, zone := range activeZones {
		activeLookup[zone] = struct{}{}
	}

	zoneListRaw, err := runFirewallCmd(conn, "--get-zones")
	if err != nil {
		return nil, fmt.Errorf("failed to list firewalld zones: %w", err)
	}
	zoneNames := strings.Fields(zoneListRaw)

	zones := make([]firewalldZoneData, 0, len(zoneNames))
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

		zones = append(zones, firewalldZoneData{
			Zone:  zoneInfo,
			Rules: parsedRules,
		})
	}

	return &firewalldData{
		DefaultZone: defaultZone,
		ActiveZones: activeZones,
		Zones:       zones,
	}, nil
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
			zone.Masquerade = firewalld.ParseBool(value)
		case "forward-ports":
			zone.ForwardPorts = splitList(value)
		case "source-ports":
			zone.SourcePorts = splitList(value)
		case "icmp-blocks":
			zone.IcmpBlocks = splitList(value)
		case "icmp-block-inversion":
			zone.IcmpBlockInversion = firewalld.ParseBool(value)
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
			switch section {
			case "source":
				inNot = true
				rule.Source.HasNot = true
			case "destination":
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
		"name":               newStringOrNil(zone.Name),
		"target":             newStringOrNil(zone.Target),
		"active":             llx.BoolData(zone.Active),
		"interfaces":         newStringArrayData(zone.Interfaces),
		"sources":            newStringArrayData(zone.Sources),
		"services":           newStringArrayData(zone.Services),
		"ports":              newStringArrayData(zone.Ports),
		"protocols":          newStringArrayData(zone.Protocols),
		"masquerade":         llx.BoolData(zone.Masquerade),
		"forwardPorts":       newStringArrayData(zone.ForwardPorts),
		"sourcePorts":        newStringArrayData(zone.SourcePorts),
		"icmpBlocks":         newStringArrayData(zone.IcmpBlocks),
		"icmpBlockInversion": llx.BoolData(zone.IcmpBlockInversion),
		"richRules":          llx.ArrayData(ruleResources, types.Resource(ResourceFirewalldRule)),
		"raw":                newStringOrNil(zone.Raw),
	}

	zoneRes, err := CreateResource(runtime, ResourceFirewalldZone, args)
	if err != nil {
		return nil, err
	}

	return zoneRes.(*mqlFirewalldZone), nil
}

func (c *mqlFirewalldZone) richRules() ([]any, error) {
	return c.RichRules.Data, c.RichRules.Error
}

func createFirewalldRuleResource(runtime *plugin.Runtime, zoneName string, idx int, rule parsedFirewalldRule) (*mqlFirewalldRule, error) {
	ruleID := fmt.Sprintf("%s/rule/%d", zoneName, idx)

	var sourceRes plugin.Resource
	var err error
	if rule.Source.HasValue || rule.Source.HasNot {
		sourceRes, err = newMqlFirewalldRuleEndpointResource(runtime, ruleID, "source", rule.Source)
		if err != nil {
			return nil, err
		}
	}

	var destRes plugin.Resource
	if rule.Dest.HasValue || rule.Dest.HasNot {
		destRes, err = newMqlFirewalldRuleEndpointResource(runtime, ruleID, "destination", rule.Dest)
		if err != nil {
			return nil, err
		}
	}

	args := map[string]*llx.RawData{
		"__id":   llx.StringData(ruleID),
		"raw":    newStringOrNil(rule.Raw),
		"family": newStringOrNil(rule.Family),
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
		args["source"] = llx.ResourceData(sourceRes, ResourceFirewalldRuleEndpoint)
	} else {
		args["source"] = llx.NilData
	}

	if destRes != nil {
		args["destination"] = llx.ResourceData(destRes, ResourceFirewalldRuleEndpoint)
	} else {
		args["destination"] = llx.NilData
	}

	ruleRes, err := CreateResource(runtime, ResourceFirewalldRule, args)
	if err != nil {
		return nil, err
	}

	return ruleRes.(*mqlFirewalldRule), nil
}

func newMqlFirewalldRuleEndpointResource(runtime *plugin.Runtime, ruleID, label string, ep parsedRuleEndpoint) (plugin.Resource, error) {
	endpointID := fmt.Sprintf("%s/%s", ruleID, label)

	args := map[string]*llx.RawData{
		"__id":    llx.StringData(endpointID),
		"address": newStringOrNil(ep.Address),
		"ipset":   newStringOrNil(ep.Ipset),
		"mac":     newStringOrNil(ep.Mac),
	}

	if ep.HasNot {
		qualifierArgs := map[string]*llx.RawData{
			"__id":    llx.StringData(endpointID + "/not"),
			"address": newStringOrNil(ep.Not.Address),
			"ipset":   newStringOrNil(ep.Not.Ipset),
			"mac":     newStringOrNil(ep.Not.Mac),
		}

		qualifierRes, err := CreateResource(runtime, ResourceFirewalldRuleEndpointQualifier, qualifierArgs)
		if err != nil {
			return nil, err
		}
		args["not"] = llx.ResourceData(qualifierRes, ResourceFirewalldRuleEndpointQualifier)
	} else {
		args["not"] = llx.NilData
	}

	res, err := CreateResource(runtime, ResourceFirewalldRuleEndpoint, args)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func newStringOrNil(s string) *llx.RawData {
	if strings.TrimSpace(s) == "" {
		return llx.NilData
	}
	return llx.StringData(s)
}

func newStringArrayData(values []string) *llx.RawData {
	return llx.ArrayData(convert.SliceAnyToInterface(values), types.String)
}
