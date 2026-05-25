// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package squid

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse_BasicDirectives(t *testing.T) {
	content := `
# This is a comment

visible_hostname proxy.example.com
http_port 3128
http_port 192.168.1.1:8080 transparent
https_port 3129 tls-cert=/etc/squid/cert.pem tls-key=/etc/squid/key.pem
via off
forwarded_for delete
`
	cfg := Parse(content)

	if got, want := cfg.Params["visible_hostname"], "proxy.example.com"; got != want {
		t.Errorf("visible_hostname = %q, want %q", got, want)
	}
	if got, want := cfg.Params["via"], "off"; got != want {
		t.Errorf("via = %q, want %q", got, want)
	}

	if got, want := len(cfg.HTTPPorts), 2; got != want {
		t.Fatalf("len(HTTPPorts) = %d, want %d", got, want)
	}
	if cfg.HTTPPorts[0].Port != 3128 || cfg.HTTPPorts[0].Address != "" {
		t.Errorf("HTTPPorts[0] = %+v", cfg.HTTPPorts[0])
	}
	if cfg.HTTPPorts[1].Port != 8080 || cfg.HTTPPorts[1].Address != "192.168.1.1" {
		t.Errorf("HTTPPorts[1] = %+v", cfg.HTTPPorts[1])
	}
	if !containsString(cfg.HTTPPorts[1].Flags, "transparent") {
		t.Errorf("HTTPPorts[1].Flags = %v, want to contain 'transparent'", cfg.HTTPPorts[1].Flags)
	}

	if got, want := len(cfg.HTTPSPorts), 1; got != want {
		t.Fatalf("len(HTTPSPorts) = %d, want %d", got, want)
	}
	l := cfg.HTTPSPorts[0]
	if l.Port != 3129 || !l.TLS || l.Cert != "/etc/squid/cert.pem" || l.Key != "/etc/squid/key.pem" {
		t.Errorf("HTTPSPorts[0] = %+v", l)
	}
}

func TestParse_IPv6Listen(t *testing.T) {
	content := `http_port [2001:db8::1]:3128 ssl-bump cert=/etc/squid/cert.pem`
	cfg := Parse(content)
	if got, want := len(cfg.HTTPPorts), 1; got != want {
		t.Fatalf("len(HTTPPorts) = %d, want %d", got, want)
	}
	l := cfg.HTTPPorts[0]
	if l.Address != "[2001:db8::1]" {
		t.Errorf("Address = %q, want %q", l.Address, "[2001:db8::1]")
	}
	if l.Port != 3128 {
		t.Errorf("Port = %d, want 3128", l.Port)
	}
	if !l.TLS {
		t.Errorf("TLS = false, want true (ssl-bump should imply TLS)")
	}
	if l.Cert != "/etc/squid/cert.pem" {
		t.Errorf("Cert = %q, want %q", l.Cert, "/etc/squid/cert.pem")
	}
	if !containsString(l.Flags, "ssl-bump") {
		t.Errorf("Flags = %v, want to contain ssl-bump", l.Flags)
	}
}

func TestParse_ACLMerge(t *testing.T) {
	content := `
acl localnet src 10.0.0.0/8
acl localnet src 172.16.0.0/12
acl localnet src 192.168.0.0/16
acl Safe_ports port 80
acl Safe_ports port 443
acl bad_url url_regex -i "/etc/squid/bad.txt"
`
	cfg := Parse(content)

	if got, want := len(cfg.ACLs), 3; got != want {
		t.Fatalf("len(ACLs) = %d, want %d", got, want)
	}

	var localnet, safePorts, badURL *ACL
	for i := range cfg.ACLs {
		switch cfg.ACLs[i].Name {
		case "localnet":
			localnet = &cfg.ACLs[i]
		case "Safe_ports":
			safePorts = &cfg.ACLs[i]
		case "bad_url":
			badURL = &cfg.ACLs[i]
		}
	}
	if localnet == nil || safePorts == nil || badURL == nil {
		t.Fatalf("missing ACLs: %+v", cfg.ACLs)
	}

	wantLocalnet := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if !reflect.DeepEqual(localnet.Values, wantLocalnet) {
		t.Errorf("localnet.Values = %v, want %v", localnet.Values, wantLocalnet)
	}
	if localnet.Type != "src" {
		t.Errorf("localnet.Type = %q, want %q", localnet.Type, "src")
	}

	wantSafe := []string{"80", "443"}
	if !reflect.DeepEqual(safePorts.Values, wantSafe) {
		t.Errorf("Safe_ports.Values = %v, want %v", safePorts.Values, wantSafe)
	}

	if !containsString(badURL.Flags, "-i") {
		t.Errorf("bad_url.Flags = %v, want to contain -i", badURL.Flags)
	}
	if len(badURL.Values) != 1 || badURL.Values[0] != "/etc/squid/bad.txt" {
		t.Errorf("bad_url.Values = %v, want [/etc/squid/bad.txt]", badURL.Values)
	}
}

func TestParse_AccessRules(t *testing.T) {
	content := `
acl localnet src 10.0.0.0/8
acl SSL_ports port 443
acl CONNECT method CONNECT

http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports
http_access allow localhost manager
http_access deny manager
http_access allow localnet
http_access allow localhost
http_access deny all

icp_access allow localnet
icp_access deny all

miss_access allow localnet
always_direct allow localnet
never_direct deny all
`
	cfg := Parse(content)

	// Filter rules per kind so we can assert order and indices.
	httpAccess := filterRules(cfg.AccessRules, "http_access")
	if got, want := len(httpAccess), 7; got != want {
		t.Fatalf("http_access count = %d, want %d", got, want)
	}
	if httpAccess[0].Action != "deny" || !reflect.DeepEqual(httpAccess[0].ACLs, []string{"!Safe_ports"}) {
		t.Errorf("http_access[0] = %+v", httpAccess[0])
	}
	if httpAccess[1].Action != "deny" || !reflect.DeepEqual(httpAccess[1].ACLs, []string{"CONNECT", "!SSL_ports"}) {
		t.Errorf("http_access[1] = %+v", httpAccess[1])
	}
	// Last rule should be "deny all".
	last := httpAccess[len(httpAccess)-1]
	if last.Action != "deny" || !reflect.DeepEqual(last.ACLs, []string{"all"}) {
		t.Errorf("last http_access = %+v, want deny all", last)
	}
	// Indices should be consecutive 0..n-1 within a kind.
	for i, r := range httpAccess {
		if r.Index != i {
			t.Errorf("http_access[%d].Index = %d, want %d", i, r.Index, i)
		}
	}

	icp := filterRules(cfg.AccessRules, "icp_access")
	if got, want := len(icp), 2; got != want {
		t.Errorf("icp_access count = %d, want %d", got, want)
	}

	if got, want := len(filterRules(cfg.AccessRules, "miss_access")), 1; got != want {
		t.Errorf("miss_access count = %d, want %d", got, want)
	}
	if got, want := len(filterRules(cfg.AccessRules, "always_direct")), 1; got != want {
		t.Errorf("always_direct count = %d, want %d", got, want)
	}
	if got, want := len(filterRules(cfg.AccessRules, "never_direct")), 1; got != want {
		t.Errorf("never_direct count = %d, want %d", got, want)
	}
}

func TestParse_CachePeer(t *testing.T) {
	content := `
cache_peer parent.example.com parent 3128 0 default no-query
cache_peer 10.0.0.1 sibling 3128 3130 proxy-only login=alice:secret
`
	cfg := Parse(content)
	if got, want := len(cfg.CachePeers), 2; got != want {
		t.Fatalf("len(CachePeers) = %d, want %d", got, want)
	}
	p := cfg.CachePeers[0]
	if p.Host != "parent.example.com" || p.Type != "parent" || p.HTTPPort != 3128 || p.ICPPort != 0 {
		t.Errorf("CachePeers[0] = %+v", p)
	}
	if !containsString(p.Options, "default") || !containsString(p.Options, "no-query") {
		t.Errorf("CachePeers[0].Options = %v", p.Options)
	}
	p = cfg.CachePeers[1]
	if p.Host != "10.0.0.1" || p.Type != "sibling" || p.HTTPPort != 3128 || p.ICPPort != 3130 {
		t.Errorf("CachePeers[1] = %+v", p)
	}
	if !containsString(p.Options, "login=alice:secret") {
		t.Errorf("CachePeers[1].Options = %v", p.Options)
	}
}

func TestParse_CacheDir(t *testing.T) {
	content := `
cache_dir ufs /var/spool/squid 100 16 256
cache_dir aufs /var/cache/squid 1000 16 256 max-size=4194304
cache_dir rock /var/spool/squid/rock 100 max-size=32768
`
	cfg := Parse(content)
	if got, want := len(cfg.CacheDirs), 3; got != want {
		t.Fatalf("len(CacheDirs) = %d, want %d", got, want)
	}

	d := cfg.CacheDirs[0]
	if d.Type != "ufs" || d.Path != "/var/spool/squid" || d.SizeMB != 100 || d.L1 != 16 || d.L2 != 256 {
		t.Errorf("CacheDirs[0] = %+v", d)
	}
	if len(d.Options) != 0 {
		t.Errorf("CacheDirs[0].Options = %v, want empty", d.Options)
	}

	d = cfg.CacheDirs[1]
	if d.Type != "aufs" || d.SizeMB != 1000 || d.L1 != 16 || d.L2 != 256 {
		t.Errorf("CacheDirs[1] = %+v", d)
	}
	if !containsString(d.Options, "max-size=4194304") {
		t.Errorf("CacheDirs[1].Options = %v", d.Options)
	}

	d = cfg.CacheDirs[2]
	if d.Type != "rock" || d.Path != "/var/spool/squid/rock" || d.SizeMB != 100 || d.L1 != 0 || d.L2 != 0 {
		t.Errorf("CacheDirs[2] = %+v", d)
	}
	if !containsString(d.Options, "max-size=32768") {
		t.Errorf("CacheDirs[2].Options = %v", d.Options)
	}
}

func TestParse_RefreshPattern(t *testing.T) {
	content := `
refresh_pattern ^ftp:           1440    20%     10080
refresh_pattern -i (/cgi-bin/|\?) 0     0%      0
refresh_pattern .               0       20%     4320 override-expire ignore-private
`
	cfg := Parse(content)
	if got, want := len(cfg.RefreshPatterns), 3; got != want {
		t.Fatalf("len(RefreshPatterns) = %d, want %d", got, want)
	}

	r := cfg.RefreshPatterns[0]
	if r.Pattern != "^ftp:" || r.Min != 1440 || r.Percent != 20 || r.Max != 10080 || r.CaseInsensitive {
		t.Errorf("RefreshPatterns[0] = %+v", r)
	}
	r = cfg.RefreshPatterns[1]
	if !r.CaseInsensitive || r.Pattern != "(/cgi-bin/|\\?)" || r.Min != 0 || r.Percent != 0 || r.Max != 0 {
		t.Errorf("RefreshPatterns[1] = %+v", r)
	}
	r = cfg.RefreshPatterns[2]
	if r.Pattern != "." || r.Max != 4320 {
		t.Errorf("RefreshPatterns[2] = %+v", r)
	}
	if !containsString(r.Options, "override-expire") || !containsString(r.Options, "ignore-private") {
		t.Errorf("RefreshPatterns[2].Options = %v", r.Options)
	}
}

func TestParse_AuthParam(t *testing.T) {
	content := `
auth_param basic program /usr/lib/squid/basic_ncsa_auth /etc/squid/passwd
auth_param basic children 5
auth_param basic realm Squid proxy-caching web server
auth_param digest program /usr/lib/squid/digest_file_auth -c /etc/squid/digestpw
`
	cfg := Parse(content)
	basic, ok := cfg.AuthParams["basic"]
	if !ok {
		t.Fatalf("missing basic auth_param: %+v", cfg.AuthParams)
	}
	if basic["program"] != "/usr/lib/squid/basic_ncsa_auth /etc/squid/passwd" {
		t.Errorf("basic.program = %q", basic["program"])
	}
	if basic["children"] != "5" {
		t.Errorf("basic.children = %q", basic["children"])
	}
	if basic["realm"] != "Squid proxy-caching web server" {
		t.Errorf("basic.realm = %q", basic["realm"])
	}
	digest, ok := cfg.AuthParams["digest"]
	if !ok {
		t.Fatalf("missing digest auth_param")
	}
	if !strings.Contains(digest["program"], "digest_file_auth") {
		t.Errorf("digest.program = %q", digest["program"])
	}
}

func TestParse_AccessLog(t *testing.T) {
	content := `
access_log /var/log/squid/access.log squid
access_log daemon:/var/log/squid/combined.log combined !manager
access_log none manager
`
	cfg := Parse(content)
	if got, want := len(cfg.AccessLogs), 3; got != want {
		t.Fatalf("len(AccessLogs) = %d, want %d", got, want)
	}
	a := cfg.AccessLogs[0]
	if a.Target != "/var/log/squid/access.log" || a.Format != "squid" || len(a.ACLs) != 0 {
		t.Errorf("AccessLogs[0] = %+v", a)
	}
	a = cfg.AccessLogs[1]
	if a.Target != "daemon:/var/log/squid/combined.log" || a.Format != "combined" {
		t.Errorf("AccessLogs[1] = %+v", a)
	}
	if !reflect.DeepEqual(a.ACLs, []string{"!manager"}) {
		t.Errorf("AccessLogs[1].ACLs = %v", a.ACLs)
	}
	a = cfg.AccessLogs[2]
	if a.Target != "none" {
		t.Errorf("AccessLogs[2].Target = %q, want %q", a.Target, "none")
	}
}

func TestParse_ContinuationLines(t *testing.T) {
	content := `
acl long_acl src \
    10.0.0.0/8 \
    172.16.0.0/12 \
    192.168.0.0/16
`
	cfg := Parse(content)
	if got, want := len(cfg.ACLs), 1; got != want {
		t.Fatalf("len(ACLs) = %d, want %d", got, want)
	}
	want := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if !reflect.DeepEqual(cfg.ACLs[0].Values, want) {
		t.Errorf("ACLs[0].Values = %v, want %v", cfg.ACLs[0].Values, want)
	}
}

func TestParse_QuotedArgs(t *testing.T) {
	content := `
auth_param basic realm "My Squid Proxy Realm"
visible_hostname "proxy host"
`
	cfg := Parse(content)
	if got, want := cfg.AuthParams["basic"]["realm"], "My Squid Proxy Realm"; got != want {
		t.Errorf("realm = %q, want %q", got, want)
	}
	if got, want := cfg.Params["visible_hostname"], "proxy host"; got != want {
		t.Errorf("visible_hostname = %q, want %q", got, want)
	}
}

func TestParseWithGlob_IncludeExpansion(t *testing.T) {
	files := map[string]string{
		"/etc/squid/squid.conf": `
visible_hostname proxy.example.com
acl localnet src 10.0.0.0/8
http_access allow localnet
include /etc/squid/conf.d/*.conf
http_access deny all
`,
		"/etc/squid/conf.d/10-extra-acls.conf": `
acl bigcorp src 203.0.113.0/24
http_access allow bigcorp
`,
		"/etc/squid/conf.d/20-cache.conf": `
cache_dir ufs /var/spool/squid 1000 16 256
refresh_pattern ^http: 0 20% 4320
`,
	}
	fileContent := func(p string) (string, error) {
		c, ok := files[p]
		if !ok {
			return "", &missingErr{path: p}
		}
		return c, nil
	}
	globExpand := func(pattern string) ([]string, error) {
		// Tiny glob expander for the fixture: `/etc/squid/conf.d/*.conf`.
		if pattern == "/etc/squid/conf.d/*.conf" {
			return []string{
				"/etc/squid/conf.d/10-extra-acls.conf",
				"/etc/squid/conf.d/20-cache.conf",
			}, nil
		}
		// Bare paths return verbatim.
		if !strings.ContainsAny(pattern, "*?[") {
			return []string{pattern}, nil
		}
		return nil, nil
	}
	cfg, err := ParseWithGlob("/etc/squid/squid.conf", fileContent, globExpand)
	if err != nil {
		t.Fatalf("ParseWithGlob: %v", err)
	}

	wantFiles := []string{
		"/etc/squid/squid.conf",
		"/etc/squid/conf.d/10-extra-acls.conf",
		"/etc/squid/conf.d/20-cache.conf",
	}
	if !reflect.DeepEqual(cfg.Files, wantFiles) {
		t.Errorf("Files = %v, want %v", cfg.Files, wantFiles)
	}

	// ACL merge across files: localnet (root) + bigcorp (snippet).
	if got, want := len(cfg.ACLs), 2; got != want {
		t.Errorf("len(ACLs) = %d, want %d", got, want)
	}

	// http_access rule order should match source order across files:
	// allow localnet (root), allow bigcorp (snippet), deny all (root).
	if got, want := len(cfg.AccessRules), 3; got != want {
		t.Fatalf("len(AccessRules) = %d, want %d", got, want)
	}
	if r := cfg.AccessRules[0]; r.Action != "allow" || r.ACLs[0] != "localnet" {
		t.Errorf("AccessRules[0] = %+v", r)
	}
	if r := cfg.AccessRules[1]; r.Action != "allow" || r.ACLs[0] != "bigcorp" {
		t.Errorf("AccessRules[1] = %+v", r)
	}
	if r := cfg.AccessRules[2]; r.Action != "deny" || r.ACLs[0] != "all" {
		t.Errorf("AccessRules[2] = %+v", r)
	}
	// Index resets per kind (all are http_access here) and runs 0,1,2.
	for i, r := range cfg.AccessRules {
		if r.Index != i {
			t.Errorf("AccessRules[%d].Index = %d, want %d", i, r.Index, i)
		}
	}

	// cache_dir and refresh_pattern came from the snippet.
	if got, want := len(cfg.CacheDirs), 1; got != want {
		t.Errorf("len(CacheDirs) = %d, want %d", got, want)
	}
	if got, want := len(cfg.RefreshPatterns), 1; got != want {
		t.Errorf("len(RefreshPatterns) = %d, want %d", got, want)
	}
}

func TestParseWithGlob_IncludeCycle(t *testing.T) {
	files := map[string]string{
		"/etc/squid/squid.conf": `
visible_hostname a
include /etc/squid/b.conf
`,
		"/etc/squid/b.conf": `
include /etc/squid/squid.conf
visible_hostname b
`,
	}
	fileContent := func(p string) (string, error) {
		c, ok := files[p]
		if !ok {
			return "", &missingErr{path: p}
		}
		return c, nil
	}
	cfg, err := ParseWithGlob("/etc/squid/squid.conf", fileContent, nil)
	if err != nil {
		t.Fatalf("ParseWithGlob: %v", err)
	}
	// Cycle protection: each file visited at most once.
	if got, want := len(cfg.Files), 2; got != want {
		t.Errorf("Files = %v, want 2 entries", cfg.Files)
	}
	// Last write wins for single-value scalars: b overrides a, but the
	// cycle stops us from visiting squid.conf again from inside b, so
	// the final value is "b".
	if got, want := cfg.Params["visible_hostname"], "b"; got != want {
		t.Errorf("visible_hostname = %q, want %q", got, want)
	}
}

func TestParse_ListenUnix(t *testing.T) {
	content := `http_port unix:/var/run/squid.sock`
	cfg := Parse(content)
	if got, want := len(cfg.HTTPPorts), 1; got != want {
		t.Fatalf("len(HTTPPorts) = %d, want %d", got, want)
	}
	l := cfg.HTTPPorts[0]
	if l.Address != "unix:/var/run/squid.sock" || l.Port != 0 {
		t.Errorf("HTTPPorts[0] = %+v", l)
	}
}

func TestParse_MultiValueParamsAreJoined(t *testing.T) {
	// http_port lines should be comma-joined in the params map (one
	// place to read the lot for quick audits).
	content := `
http_port 3128
http_port 8080
`
	cfg := Parse(content)
	if got, want := cfg.Params["http_port"], "3128,8080"; got != want {
		t.Errorf("params[http_port] = %q, want %q", got, want)
	}
}

// ----------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------

func filterRules(rules []AccessRule, kind string) []AccessRule {
	var out []AccessRule
	for _, r := range rules {
		if r.Kind == kind {
			out = append(out, r)
		}
	}
	return out
}

type missingErr struct{ path string }

func (e *missingErr) Error() string { return "missing: " + e.path }
