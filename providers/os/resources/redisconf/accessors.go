// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package redisconf

import (
	"strings"
)

// Server defaults that apply when a directive is absent. They are named
// because several are permissive or surprising, and a zero-value fallback
// would misreport them.
const (
	DefaultPort            = 6379
	DefaultTLSAuthClients  = "yes"
	DefaultMaxmemoryPolicy = "noeviction"
	DefaultAppendFsync     = "everysec"
	DefaultDir             = "./"
	DefaultDbFilename      = "dump.rdb"
	DefaultACLPubsub       = "resetchannels"
)

// LastOfAny reports the arguments of the last occurrence of any of the given
// directive names, resolved by document order rather than by preferring one
// spelling.
//
// Valkey renamed several directives and kept the Redis names as aliases
// (primaryauth for masterauth, primaryuser for masteruser). Both are live, so
// a reader that checked one name first would report the wrong value on a file
// that sets the other, and one that scanned per name would ignore which came
// last.
func (c *Conf) LastOfAny(names ...string) []string {
	var out []string
	found := false
	for _, d := range c.Directives {
		for _, name := range names {
			if strings.EqualFold(d.Name, name) {
				out = d.Args
				found = true
				break
			}
		}
	}
	if !found {
		return nil
	}
	if out == nil {
		return []string{}
	}
	return out
}

// ---------------------------------------------------------------------------
// network exposure
// ---------------------------------------------------------------------------

// Bind reports the configured bind addresses, with the leading `-` that marks
// an address the server may skip stripped off.
func (c *Conf) Bind() []string {
	args := c.Last("bind")
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, strings.TrimPrefix(a, "-"))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// wildcardBinds are the addresses that accept connections on every interface.
var wildcardBinds = map[string]bool{
	"0.0.0.0": true,
	"*":       true,
	"::":      true,
	"::0":     true,
	"::*":     true,
}

// BindsAllInterfaces reports whether the server accepts connections on every
// interface.
//
// The absent case is the one that matters and it is not the safe one: with no
// bind directive at all the server listens on every interface. A reader that
// treated "no bind configured" as "bound to loopback" would report the most
// exposed configuration as the least.
func (c *Conf) BindsAllInterfaces() bool {
	addrs := c.Bind()
	if len(addrs) == 0 {
		return true
	}
	for _, a := range addrs {
		if wildcardBinds[a] {
			return true
		}
	}
	return false
}

// ProtectedMode reports whether protected mode is on, which is the default.
//
// Protected mode is what keeps a server with no bind and no password from
// serving the network, so it is the backstop for BindsAllInterfaces. Turning
// it off without setting a password or an explicit bind opens the server.
func (c *Conf) ProtectedMode() bool {
	return c.Bool("protected-mode", true)
}

// Port reports the plaintext TCP port, which is 0 when the plaintext listener
// is switched off.
func (c *Conf) Port() int64 {
	return c.Int("port", DefaultPort)
}

// RequirepassSet reports whether a password is configured, without exposing
// the password itself.
func (c *Conf) RequirepassSet() bool {
	return c.String("requirepass", "") != ""
}

// ReplicationAuthSet reports whether this server authenticates to its
// replication source, reading both the Redis and the Valkey spelling.
func (c *Conf) ReplicationAuthSet() bool {
	args := c.LastOfAny("masterauth", "primaryauth")
	return len(args) > 0 && args[0] != ""
}

// ReplicationUser reports the replication username, reading both spellings.
func (c *Conf) ReplicationUser() string {
	args := c.LastOfAny("masteruser", "primaryuser")
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// UnixSocket reports the unix socket path, empty when none is configured.
func (c *Conf) UnixSocket() string {
	return c.String("unixsocket", "")
}

// UnixSocketPerm reports the socket mode as written, for example "700".
func (c *Conf) UnixSocketPerm() string {
	return c.String("unixsocketperm", "")
}

// ---------------------------------------------------------------------------
// TLS
// ---------------------------------------------------------------------------

// TLSPort reports the TLS port, 0 when TLS is not enabled.
func (c *Conf) TLSPort() int64 {
	return c.Int("tls-port", 0)
}

// TLSEnabled reports whether the TLS listener is on.
func (c *Conf) TLSEnabled() bool {
	return c.TLSPort() > 0
}

// TLSAuthClients reports the client-certificate policy: yes, no, or optional.
//
// The default is yes, so a TLS-enabled server requires client certificates
// unless the file says otherwise. Defaulting this to no would report mutual
// TLS as absent on a server that enforces it.
func (c *Conf) TLSAuthClients() string {
	return c.String("tls-auth-clients", DefaultTLSAuthClients)
}

// ---------------------------------------------------------------------------
// command surface
// ---------------------------------------------------------------------------

// RenamedCommands maps each renamed command to its replacement name, with an
// empty value meaning the command is disabled outright.
//
// Renaming or disabling CONFIG is common hardening, and it is also what stops
// the redisdb provider from reading the running configuration, which is part
// of why this file is worth reading directly.
func (c *Conf) RenamedCommands() map[string]string {
	out := map[string]string{}
	for _, args := range c.All("rename-command") {
		if len(args) < 2 {
			continue
		}
		out[strings.ToUpper(args[0])] = args[1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DisabledCommands lists the commands renamed to the empty string, which
// removes them entirely.
func (c *Conf) DisabledCommands() []string {
	var out []string
	for name, to := range c.RenamedCommands() {
		if to == "" {
			out = append(out, name)
		}
	}
	return out
}

// protectedConfigValue reads one of the enable-* directives, whose accepted
// values are no, yes, and local.
func (c *Conf) protectedConfigValue(name string) string {
	return c.String(name, "no")
}

// EnableProtectedConfigs reports whether the protected CONFIG parameters can
// be changed at runtime: no, yes, or local.
func (c *Conf) EnableProtectedConfigs() string {
	return c.protectedConfigValue("enable-protected-configs")
}

// EnableDebugCommand reports whether the DEBUG command is available: no, yes,
// or local. DEBUG can crash or stall the server.
func (c *Conf) EnableDebugCommand() string {
	return c.protectedConfigValue("enable-debug-command")
}

// EnableModuleCommand reports whether MODULE is available: no, yes, or local.
// MODULE LOAD runs native code inside the server process.
func (c *Conf) EnableModuleCommand() string {
	return c.protectedConfigValue("enable-module-command")
}

// ---------------------------------------------------------------------------
// persistence
// ---------------------------------------------------------------------------

// SavePoint is one RDB snapshot rule.
type SavePoint struct {
	Seconds int64
	Changes int64
}

// SavePoints reports the RDB snapshot schedule.
//
// Three behaviors combine here. A save line may carry several pairs, every
// save line adds to the schedule rather than replacing it, and `save ""`
// clears everything configured so far. On top of that, a file with no save
// directive at all gets the built-in schedule rather than no snapshots, so
// absent has to report the default rather than empty.
func (c *Conf) SavePoints() []SavePoint {
	if !c.Has("save") {
		return []SavePoint{{3600, 1}, {300, 100}, {60, 10000}}
	}

	var out []SavePoint
	for _, args := range c.All("save") {
		// `save ""` resets the schedule.
		if len(args) == 1 && args[0] == "" {
			out = nil
			continue
		}
		for i := 0; i+1 < len(args); i += 2 {
			secs, err1 := parseInt(args[i])
			changes, err2 := parseInt(args[i+1])
			if err1 || err2 {
				continue
			}
			out = append(out, SavePoint{secs, changes})
		}
	}
	return out
}

// RDBEnabled reports whether RDB snapshotting is on, which it is unless the
// schedule was explicitly cleared.
func (c *Conf) RDBEnabled() bool {
	return len(c.SavePoints()) > 0
}

// AppendOnly reports whether append-only file persistence is on.
func (c *Conf) AppendOnly() bool {
	return c.Bool("appendonly", false)
}

// AppendFsync reports the append-only fsync policy: everysec, always, or no.
func (c *Conf) AppendFsync() string {
	return c.String("appendfsync", DefaultAppendFsync)
}

func parseInt(s string) (int64, bool) {
	var n int64
	if s == "" {
		return 0, true
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, true
		}
		n = n*10 + int64(s[i]-'0')
	}
	return n, false
}

// ---------------------------------------------------------------------------
// access control
// ---------------------------------------------------------------------------

// ACLUser is one access-control user defined inline in the configuration.
type ACLUser struct {
	Name            string
	IsDefault       bool
	Enabled         bool
	Nopass          bool
	PasswordCount   int64
	KeyPatterns     []string
	ChannelPatterns []string
	CommandRules    []string
}

// ACLFile reports the external ACL file path, empty when the rules are inline.
func (c *Conf) ACLFile() string {
	return c.String("aclfile", "")
}

// ACLPubsubDefault reports the default channel permission for new users:
// resetchannels (the default, no channels) or allchannels.
func (c *Conf) ACLPubsubDefault() string {
	return c.String("acl-pubsub-default", DefaultACLPubsub)
}

// ACLUsers reports the users defined by inline `user` directives.
//
// A user is disabled unless a rule turns it on, so Enabled defaults to false.
// The rules are order-sensitive: `resetpass` drops the passwords collected so
// far and `resetchannels` drops the channels, so a later rule can undo an
// earlier one and the scan has to apply them in sequence.
func (c *Conf) ACLUsers() []ACLUser {
	var out []ACLUser
	for _, args := range c.All("user") {
		if len(args) == 0 {
			continue
		}
		u := ACLUser{Name: args[0]}
		u.IsDefault = u.Name == "default"

		for _, rule := range args[1:] {
			switch {
			case strings.EqualFold(rule, "on"):
				u.Enabled = true
			case strings.EqualFold(rule, "off"):
				u.Enabled = false
			case strings.EqualFold(rule, "nopass"):
				u.Nopass = true
				u.PasswordCount = 0
			case strings.EqualFold(rule, "resetpass"):
				u.Nopass = false
				u.PasswordCount = 0
			case strings.HasPrefix(rule, ">"), strings.HasPrefix(rule, "#"):
				u.PasswordCount++
				u.Nopass = false
			case strings.EqualFold(rule, "allkeys"):
				u.KeyPatterns = append(u.KeyPatterns, "*")
			case strings.EqualFold(rule, "resetkeys"):
				u.KeyPatterns = nil
			case strings.HasPrefix(rule, "~"):
				u.KeyPatterns = append(u.KeyPatterns, strings.TrimPrefix(rule, "~"))
			case strings.HasPrefix(rule, "%"):
				// Read- and write-scoped key patterns (%R~, %W~, %RW~) keep
				// their prefix, since dropping it would report a read-only
				// grant as full access.
				u.KeyPatterns = append(u.KeyPatterns, rule)
			case strings.EqualFold(rule, "allchannels"):
				u.ChannelPatterns = append(u.ChannelPatterns, "*")
			case strings.EqualFold(rule, "resetchannels"):
				u.ChannelPatterns = nil
			case strings.HasPrefix(rule, "&"):
				u.ChannelPatterns = append(u.ChannelPatterns, strings.TrimPrefix(rule, "&"))
			case strings.EqualFold(rule, "allcommands"):
				u.CommandRules = append(u.CommandRules, "+@all")
			case strings.EqualFold(rule, "nocommands"):
				u.CommandRules = append(u.CommandRules, "-@all")
			case strings.HasPrefix(rule, "+"), strings.HasPrefix(rule, "-"):
				u.CommandRules = append(u.CommandRules, rule)
			}
		}
		out = append(out, u)
	}
	return out
}

// ---------------------------------------------------------------------------
// flavor
// ---------------------------------------------------------------------------

// valkeyOnlyDirectives are settings Valkey added that Redis has never had, so
// their presence identifies the file as Valkey's regardless of its path.
var valkeyOnlyDirectives = []string{
	"availability-zone",
	"dual-channel-replication-enabled",
	"extended-redis-compatibility",
	"primaryauth",
	"primaryuser",
	"cluster-announce-client-ipv4",
	"cluster-announce-client-ipv6",
	"commandlog-slow-execution-max-len",
	"cluster-slot-stats-enabled",
}

// IsValkey reports whether the file is a Valkey configuration.
//
// It keys off directives Valkey introduced rather than off the file path, so
// a Valkey install that reuses the redis.conf name is still identified. A
// file that sets none of them is reported as Redis, which is the right way
// round: the two share the format, so the Redis reading stays correct either
// way.
func (c *Conf) IsValkey() bool {
	for _, name := range valkeyOnlyDirectives {
		if c.Has(name) {
			return true
		}
	}
	return false
}
