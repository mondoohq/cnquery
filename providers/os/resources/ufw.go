// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

const (
	ufwConfPath     = "/etc/ufw/ufw.conf"
	ufwDefaultsPath = "/etc/default/ufw"
	ufwRulesPath    = "/etc/ufw/user.rules"
	ufwRules6Path   = "/etc/ufw/user6.rules"
)

type mqlUfwInternal struct {
	fetched          bool
	cacheStatus      string
	cacheDefIncoming string
	cacheDefOutgoing string
	cacheDefRouted   string
	cacheLogging     string
	lock             sync.Mutex
}

func (u *mqlUfw) id() (string, error) {
	return "ufw", nil
}

func (u *mqlUfw) getFs() (afero.Afero, error) {
	conn, ok := u.MqlRuntime.Connection.(shared.Connection)
	if !ok || !conn.Capabilities().Has(shared.Capability_File) {
		return afero.Afero{}, fmt.Errorf("ufw requires file system capability")
	}
	return afero.Afero{Fs: conn.FileSystem()}, nil
}

func (u *mqlUfw) fetchStatus() error {
	if u.fetched {
		return nil
	}
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.fetched {
		return nil
	}

	afs, err := u.getFs()
	if err != nil {
		return err
	}

	// Read /etc/ufw/ufw.conf for ENABLED and LOGLEVEL
	confData, err := afs.ReadFile(ufwConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			u.cacheStatus = "not installed"
			u.fetched = true
			return nil
		}
		return err
	}

	conf := parseUfwKeyValue(string(confData))
	if strings.EqualFold(conf["ENABLED"], "yes") {
		u.cacheStatus = "active"
	} else {
		u.cacheStatus = "inactive"
	}
	u.cacheLogging = strings.ToLower(conf["LOGLEVEL"])

	// Read /etc/default/ufw for default policies
	defaultsData, err := afs.ReadFile(ufwDefaultsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		defaults := parseUfwKeyValue(string(defaultsData))
		u.cacheDefIncoming = ufwPolicyName(defaults["DEFAULT_INPUT_POLICY"])
		u.cacheDefOutgoing = ufwPolicyName(defaults["DEFAULT_OUTPUT_POLICY"])
		u.cacheDefRouted = ufwPolicyName(defaults["DEFAULT_FORWARD_POLICY"])
	}

	u.fetched = true
	return nil
}

func (u *mqlUfw) status() (string, error) {
	if err := u.fetchStatus(); err != nil {
		return "", err
	}
	return u.cacheStatus, nil
}

func (u *mqlUfw) defaultIncoming() (string, error) {
	if err := u.fetchStatus(); err != nil {
		return "", err
	}
	return u.cacheDefIncoming, nil
}

func (u *mqlUfw) defaultOutgoing() (string, error) {
	if err := u.fetchStatus(); err != nil {
		return "", err
	}
	return u.cacheDefOutgoing, nil
}

func (u *mqlUfw) defaultRouted() (string, error) {
	if err := u.fetchStatus(); err != nil {
		return "", err
	}
	return u.cacheDefRouted, nil
}

func (u *mqlUfw) logging() (string, error) {
	if err := u.fetchStatus(); err != nil {
		return "", err
	}
	return u.cacheLogging, nil
}

func (u *mqlUfw) rules() ([]any, error) {
	if err := u.fetchStatus(); err != nil {
		return nil, err
	}
	if u.cacheStatus == "not installed" {
		return []any{}, nil
	}

	afs, err := u.getFs()
	if err != nil {
		return nil, err
	}

	var rules []any
	num := int64(1)

	// Parse IPv4 rules
	v4Data, err := afs.ReadFile(ufwRulesPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		tuples := parseUfwTuples(string(v4Data))
		for _, t := range tuples {
			t.number = num
			t.ipv6 = false
			res, err := createUfwRuleResource(u.MqlRuntime, t)
			if err != nil {
				return nil, err
			}
			rules = append(rules, res)
			num++
		}
	}

	// Parse IPv6 rules
	v6Data, err := afs.ReadFile(ufwRules6Path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		tuples := parseUfwTuples(string(v6Data))
		for _, t := range tuples {
			t.number = num
			t.ipv6 = true
			res, err := createUfwRuleResource(u.MqlRuntime, t)
			if err != nil {
				return nil, err
			}
			rules = append(rules, res)
			num++
		}
	}

	return rules, nil
}

const ufwAppsDir = "/etc/ufw/applications.d"

func (u *mqlUfw) applications() ([]any, error) {
	if err := u.fetchStatus(); err != nil {
		return nil, err
	}
	if u.cacheStatus == "not installed" {
		return []any{}, nil
	}

	afs, err := u.getFs()
	if err != nil {
		return nil, err
	}

	entries, err := afs.ReadDir(ufwAppsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var apps []any
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := afs.ReadFile(ufwAppsDir + "/" + entry.Name())
		if err != nil {
			continue
		}
		parsed := parseUfwApplications(string(data))
		for _, app := range parsed {
			res, err := CreateResource(u.MqlRuntime, "ufw.application", map[string]*llx.RawData{
				"name":        llx.StringData(app.name),
				"title":       llx.StringData(app.title),
				"description": llx.StringData(app.description),
				"ports":       llx.StringData(app.ports),
			})
			if err != nil {
				return nil, err
			}
			apps = append(apps, res)
		}
	}
	return apps, nil
}

func (a *mqlUfwApplication) id() (string, error) {
	return "ufw/application/" + a.Name.Data, nil
}

type ufwParsedApp struct {
	name        string
	title       string
	description string
	ports       string
}

// parseUfwApplications parses a UFW application profile file (INI-like format).
//
// Example:
//
//	[Nginx Full]
//	title=Web Server (Nginx, HTTP + HTTPS)
//	description=Small, but very powerful and efficient web server
//	ports=80,443/tcp
func parseUfwApplications(data string) []ufwParsedApp {
	var apps []ufwParsedApp
	var current *ufwParsedApp

	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				apps = append(apps, *current)
			}
			current = &ufwParsedApp{
				name: line[1 : len(line)-1],
			}
			continue
		}
		if current == nil {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "title":
			current.title = strings.TrimSpace(val)
		case "description":
			current.description = strings.TrimSpace(val)
		case "ports":
			current.ports = strings.TrimSpace(val)
		}
	}
	if current != nil {
		apps = append(apps, *current)
	}
	return apps
}

func createUfwRuleResource(runtime *plugin.Runtime, rule ufwParsedRule) (plugin.Resource, error) {
	return CreateResource(runtime, "ufw.rule", map[string]*llx.RawData{
		"number":    llx.IntData(rule.number),
		"action":    llx.StringData(rule.action),
		"direction": llx.StringData(rule.direction),
		"protocol":  llx.StringData(rule.protocol),
		"port":      llx.StringData(rule.port),
		"interface": llx.StringData(rule.iface),
		"from":      llx.StringData(rule.from),
		"to":        llx.StringData(rule.to),
		"ipv6":      llx.BoolData(rule.ipv6),
		"raw":       llx.StringData(rule.raw),
	})
}

func (r *mqlUfwRule) id() (string, error) {
	// Use the raw tuple line as the ID so that caching is stable even if
	// rules are reordered or new rules are inserted between invocations.
	return "ufw/rule/" + r.Raw.Data, nil
}

// parseUfwKeyValue parses a simple KEY=VALUE config file (with optional quotes).
// Lines starting with # are comments.
func parseUfwKeyValue(data string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), "\"'")
		result[strings.TrimSpace(key)] = val
	}
	return result
}

// ufwPolicyName translates iptables policy names from /etc/default/ufw
// to UFW-style names that users expect (e.g., DROP -> deny, ACCEPT -> allow).
func ufwPolicyName(raw string) string {
	switch strings.ToLower(raw) {
	case "drop":
		return "deny"
	case "accept":
		return "allow"
	default:
		return strings.ToLower(raw)
	}
}

type ufwParsedRule struct {
	number    int64
	action    string
	direction string
	protocol  string
	port      string
	iface     string
	from      string
	to        string
	ipv6      bool
	raw       string
}

// parseUfwTuples parses UFW tuple comments from a user.rules or user6.rules file.
//
// Tuple format:
//
//	### tuple ### ACTION PROTOCOL DPORT DST SPORT SRC DIRECTION
//	### tuple ### ACTION PROTOCOL DPORT DST SPORT SRC DAPP SAPP DIRECTION
//
// Direction is always the last field and may include an interface: "in", "out", "in_eth0".
//
// Examples:
//
//	### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
//	### tuple ### deny tcp 3306 0.0.0.0/0 any 0.0.0.0/0 in
//	### tuple ### allow tcp 443 0.0.0.0/0 any 10.0.0.0/8 in_eth0
//	### tuple ### limit tcp 22 0.0.0.0/0 any 0.0.0.0/0 in
func parseUfwTuples(data string) []ufwParsedRule {
	var rules []ufwParsedRule
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "### tuple ###") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "### tuple ###"))
		fields := strings.Fields(rest)
		if len(fields) < 7 {
			continue
		}

		rule := ufwParsedRule{raw: line}

		// Action is first field, may have _log or _log-all suffix
		action := fields[0]
		if i := strings.Index(action, "_log"); i != -1 {
			action = action[:i]
		}
		rule.action = strings.ToUpper(action)

		rule.protocol = fields[1]
		rule.port = fields[2]
		rule.to = fields[3]
		// fields[4] is sport (source port), usually "any"
		rule.from = fields[5]

		// Direction is always the last field
		dirField := fields[len(fields)-1]
		dir, iface, hasIface := strings.Cut(dirField, "_")
		rule.direction = strings.ToUpper(dir)
		if hasIface {
			rule.iface = iface
		}

		rules = append(rules, rule)
	}
	return rules
}
