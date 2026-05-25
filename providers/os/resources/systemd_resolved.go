// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlSystemdResolved) id() (string, error) {
	return "systemd.resolved", nil
}

func (r *mqlSystemdResolved) active() (bool, error) {
	return isSystemdUnitActive(r.MqlRuntime, "systemd-resolved"), nil
}

type resolvedGlobal struct {
	dns            []string
	fallbackDns    []string
	domains        []string
	dnssec         string
	dnsOverTls     string
	llmnr          string
	multicastDns   string
	resolvConfMode string
	cache          bool
}

func (r *mqlSystemdResolved) resolveGlobal() (*resolvedGlobal, error) {
	if r.cachedGlobal != nil {
		return r.cachedGlobal, nil
	}
	stdout, ok, err := runSystemctl(r.MqlRuntime, "resolvectl status --no-pager")
	if err != nil {
		return nil, err
	}
	g := &resolvedGlobal{}
	if ok {
		parseResolvectlGlobal(stdout, g)
	}
	r.cachedGlobal = g
	return g, nil
}

func (r *mqlSystemdResolved) dns() ([]any, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return nil, err
	}
	return stringsToAny(g.dns), nil
}

func (r *mqlSystemdResolved) fallbackDns() ([]any, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return nil, err
	}
	return stringsToAny(g.fallbackDns), nil
}

func (r *mqlSystemdResolved) domains() ([]any, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return nil, err
	}
	return stringsToAny(g.domains), nil
}

func (r *mqlSystemdResolved) dnssec() (string, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return "", err
	}
	return g.dnssec, nil
}

func (r *mqlSystemdResolved) dnsOverTls() (string, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return "", err
	}
	return g.dnsOverTls, nil
}

func (r *mqlSystemdResolved) llmnr() (string, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return "", err
	}
	return g.llmnr, nil
}

func (r *mqlSystemdResolved) multicastDns() (string, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return "", err
	}
	return g.multicastDns, nil
}

func (r *mqlSystemdResolved) resolvConfMode() (string, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return "", err
	}
	return g.resolvConfMode, nil
}

func (r *mqlSystemdResolved) cache() (bool, error) {
	g, err := r.resolveGlobal()
	if err != nil {
		return false, err
	}
	return g.cache, nil
}

type mqlSystemdResolvedInternal struct {
	cachedGlobal *resolvedGlobal
}

// parseResolvectlGlobal extracts the Global-scope fields from
// `resolvectl status` output. The Global block ends at the first blank
// line or at the first "Link N (name)" header. Field lines are
// right-aligned by resolvectl; we strip leading whitespace.
//
// Example block:
//
//	Global
//	         Protocols: -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
//	  resolv.conf mode: stub
//	Current DNS Server: 1.1.1.1
//	       DNS Servers: 1.1.1.1 1.0.0.1
//	        DNS Domain: corp.example.com ~example.com
func parseResolvectlGlobal(stdout string, g *resolvedGlobal) {
	// Default cache to true — systemd-resolved caches by default, and the
	// `Cache` line only appears in some versions. The `Protocols` line is
	// where modern resolvectl puts the cache state.
	g.cache = true

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	inGlobal := false
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "Global" {
			inGlobal = true
			continue
		}
		if !inGlobal {
			continue
		}
		// Global block ends at first blank line or any "Link N" header.
		if trimmed == "" || strings.HasPrefix(trimmed, "Link ") {
			return
		}

		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		value := strings.TrimSpace(trimmed[idx+1:])

		switch key {
		case "Protocols":
			parseResolvectlProtocols(value, g)
		case "resolv.conf mode":
			g.resolvConfMode = value
		case "DNS Servers":
			g.dns = strings.Fields(value)
		case "Fallback DNS Servers":
			g.fallbackDns = strings.Fields(value)
		case "DNS Domain":
			g.domains = strings.Fields(value)
		}
	}
}

// parseResolvectlProtocols parses the `Protocols:` line from `resolvectl
// status`. The format is a space-separated list of `+FLAG`/`-FLAG`
// tokens followed by `KEY=VALUE` pairs:
//
//	-LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
//
// We translate the +/- prefixes into the same `yes`/`no` string vocabulary
// the corresponding configuration directives use.
func parseResolvectlProtocols(value string, g *resolvedGlobal) {
	for _, tok := range strings.Fields(value) {
		if eq := strings.Index(tok, "="); eq > 0 {
			k := tok[:eq]
			v := tok[eq+1:]
			switch k {
			case "DNSSEC":
				g.dnssec = v
			case "DNSOverTLS":
				g.dnsOverTls = v
			case "LLMNR":
				g.llmnr = v
			case "MulticastDNS":
				g.multicastDns = v
			}
			continue
		}
		if len(tok) < 2 {
			continue
		}
		state := "no"
		if tok[0] == '+' {
			state = "yes"
		}
		switch tok[1:] {
		case "LLMNR":
			g.llmnr = state
		case "mDNS":
			g.multicastDns = state
		case "DNSOverTLS":
			g.dnsOverTls = state
		}
	}
}

// isSystemdUnitActive returns true if `systemctl is-active <unit>` exits 0.
// Any other outcome (binary missing, unit not loaded, inactive, failed)
// returns false — we deliberately don't distinguish those at the resource
// level; downstream queries can inspect the unit directly.
func isSystemdUnitActive(runtime *plugin.Runtime, unit string) bool {
	o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData("systemctl is-active " + shellQuoteUnit(unit)),
	})
	if err != nil {
		return false
	}
	cmd := o.(*mqlCommand)
	return cmd.GetExitcode().Data == 0
}
