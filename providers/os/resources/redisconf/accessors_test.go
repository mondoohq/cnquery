// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package redisconf_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers/os/resources/redisconf"
)

// An empty file is a server running entirely on built-in defaults, and
// several of those defaults are the exposed ones.
func TestDefaultsOfAnEmptyFile(t *testing.T) {
	c := load(t, map[string]string{"/c": ""}, "/c")

	assert.True(t, c.BindsAllInterfaces(), "no bind directive means every interface")
	assert.True(t, c.ProtectedMode(), "protected mode is on unless turned off")
	assert.False(t, c.RequirepassSet())
	assert.Equal(t, int64(6379), c.Port())
	assert.False(t, c.TLSEnabled())
	assert.Equal(t, "yes", c.TLSAuthClients(), "client certs are required by default")
	assert.True(t, c.RDBEnabled(), "snapshots run on the built-in schedule")
	assert.False(t, c.AppendOnly())
	assert.Equal(t, "noeviction", c.String("maxmemory-policy", redisconf.DefaultMaxmemoryPolicy))
	assert.Equal(t, "no", c.EnableDebugCommand())
}

// The absent case is the exposed one, which is the opposite of what a
// zero-value default would report.
func TestBindsAllInterfaces(t *testing.T) {
	for _, tc := range []struct {
		name     string
		conf     string
		expected bool
	}{
		{"absent", "port 6379\n", true},
		{"loopback only", "bind 127.0.0.1 -::1\n", false},
		{"ipv4 wildcard", "bind 0.0.0.0\n", true},
		{"ipv6 wildcard", "bind ::\n", true},
		{"star", "bind *\n", true},
		{"star with loopback", "bind 127.0.0.1 *\n", true},
		{"specific address", "bind 10.0.0.5\n", false},
		{"optional marker stripped", "bind -0.0.0.0\n", true},
		// A later bind replaces an earlier one rather than adding to it.
		{"last bind wins", "bind 0.0.0.0\nbind 127.0.0.1\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := load(t, map[string]string{"/c": tc.conf}, "/c")
			assert.Equal(t, tc.expected, c.BindsAllInterfaces())
		})
	}
}

// save accumulates across lines, a single line may carry several pairs, and
// `save ""` clears the schedule.
func TestSavePoints(t *testing.T) {
	for _, tc := range []struct {
		name     string
		conf     string
		expected []redisconf.SavePoint
		rdb      bool
	}{
		{
			"absent uses the built-in schedule",
			"port 6379\n",
			[]redisconf.SavePoint{{3600, 1}, {300, 100}, {60, 10000}},
			true,
		},
		{
			"multiple pairs on one line",
			`save 3600 1 300 100 60 10000` + "\n",
			[]redisconf.SavePoint{{3600, 1}, {300, 100}, {60, 10000}},
			true,
		},
		{
			"lines accumulate",
			"save 900 1\nsave 300 10\n",
			[]redisconf.SavePoint{{900, 1}, {300, 10}},
			true,
		},
		{
			"empty string clears",
			`save ""` + "\n",
			nil,
			false,
		},
		{
			"clear then re-add",
			"save 900 1\nsave \"\"\nsave 60 100\n",
			[]redisconf.SavePoint{{60, 100}},
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := load(t, map[string]string{"/c": tc.conf}, "/c")
			assert.Equal(t, tc.expected, c.SavePoints())
			assert.Equal(t, tc.rdb, c.RDBEnabled())
		})
	}
}

// Valkey renamed these and kept the Redis spelling as an alias, so reading
// only one name reports replication auth as absent on half the fleet.
func TestReplicationAuthReadsBothSpellings(t *testing.T) {
	redis := load(t, map[string]string{"/c": "masterauth secret\nmasteruser repl\n"}, "/c")
	assert.True(t, redis.ReplicationAuthSet())
	assert.Equal(t, "repl", redis.ReplicationUser())

	valkey := load(t, map[string]string{"/c": "primaryauth secret\nprimaryuser repl\n"}, "/c")
	assert.True(t, valkey.ReplicationAuthSet())
	assert.Equal(t, "repl", valkey.ReplicationUser())

	none := load(t, map[string]string{"/c": "port 6379\n"}, "/c")
	assert.False(t, none.ReplicationAuthSet())
	assert.Empty(t, none.ReplicationUser())
}

// Aliases resolve by document order, so the last one written wins regardless
// of which spelling it uses.
func TestAliasResolvesByDocumentOrder(t *testing.T) {
	c := load(t, map[string]string{"/c": "masteruser old\nprimaryuser new\n"}, "/c")
	assert.Equal(t, "new", c.ReplicationUser())

	reversed := load(t, map[string]string{"/c": "primaryuser new\nmasteruser old\n"}, "/c")
	assert.Equal(t, "old", reversed.ReplicationUser())
}

func TestRenamedAndDisabledCommands(t *testing.T) {
	c := load(t, map[string]string{"/c": `
rename-command CONFIG b840fc02d524045429941cc15f59e41cb7be6c52
rename-command FLUSHALL ""
rename-command DEBUG ""
`}, "/c")

	assert.Equal(t, map[string]string{
		"CONFIG":   "b840fc02d524045429941cc15f59e41cb7be6c52",
		"FLUSHALL": "",
		"DEBUG":    "",
	}, c.RenamedCommands())
	assert.ElementsMatch(t, []string{"FLUSHALL", "DEBUG"}, c.DisabledCommands())
}

func TestACLUsers(t *testing.T) {
	c := load(t, map[string]string{"/c": `
user default on nopass ~* &* +@all
user worker on >secret ~jobs:* +@list +@connection
user disabled off ~* +@all
user reader on #a1b2 %R~cache:* +@read
`}, "/c")

	users := c.ACLUsers()
	assert.Len(t, users, 4)

	byName := map[string]redisconf.ACLUser{}
	for _, u := range users {
		byName[u.Name] = u
	}

	// The riskiest shape: enabled, no password, all keys, all commands.
	def := byName["default"]
	assert.True(t, def.IsDefault)
	assert.True(t, def.Enabled)
	assert.True(t, def.Nopass)
	assert.Equal(t, []string{"*"}, def.KeyPatterns)
	assert.Equal(t, []string{"*"}, def.ChannelPatterns)
	assert.Equal(t, []string{"+@all"}, def.CommandRules)

	worker := byName["worker"]
	assert.True(t, worker.Enabled)
	assert.False(t, worker.Nopass)
	assert.Equal(t, int64(1), worker.PasswordCount)
	assert.Equal(t, []string{"jobs:*"}, worker.KeyPatterns)
	assert.Equal(t, []string{"+@list", "+@connection"}, worker.CommandRules)

	assert.False(t, byName["disabled"].Enabled)

	// A read-scoped pattern keeps its prefix, since dropping it would report
	// a read-only grant as full access.
	reader := byName["reader"]
	assert.Equal(t, int64(1), reader.PasswordCount)
	assert.Equal(t, []string{"%R~cache:*"}, reader.KeyPatterns)
}

// ACL rules are order-sensitive: a later reset undoes an earlier grant.
func TestACLUserRuleOrdering(t *testing.T) {
	c := load(t, map[string]string{"/c": `
user a on >first >second resetpass >third
user b on nopass >withpass
user c on allchannels resetchannels &only:this
user d on ~* resetkeys ~narrow:*
`}, "/c")

	byName := map[string]redisconf.ACLUser{}
	for _, u := range c.ACLUsers() {
		byName[u.Name] = u
	}

	assert.Equal(t, int64(1), byName["a"].PasswordCount, "resetpass drops the earlier two")
	assert.False(t, byName["b"].Nopass, "a password after nopass clears it")
	assert.Equal(t, []string{"only:this"}, byName["c"].ChannelPatterns)
	assert.Equal(t, []string{"narrow:*"}, byName["d"].KeyPatterns)
}

// A user with no explicit `on` cannot authenticate, so enabled must not
// default to true.
func TestACLUserDisabledByDefault(t *testing.T) {
	c := load(t, map[string]string{"/c": "user lurker ~* +@all\n"}, "/c")
	users := c.ACLUsers()
	assert.Len(t, users, 1)
	assert.False(t, users[0].Enabled)
}

func TestProtectedConfigToggles(t *testing.T) {
	c := load(t, map[string]string{"/c": `
enable-protected-configs yes
enable-debug-command local
`}, "/c")

	assert.Equal(t, "yes", c.EnableProtectedConfigs())
	assert.Equal(t, "local", c.EnableDebugCommand())
	assert.Equal(t, "no", c.EnableModuleCommand(), "unset stays off")
}

func TestTLS(t *testing.T) {
	off := load(t, map[string]string{"/c": "port 6379\n"}, "/c")
	assert.False(t, off.TLSEnabled())
	assert.Equal(t, int64(0), off.TLSPort())

	on := load(t, map[string]string{"/c": "port 0\ntls-port 6379\ntls-auth-clients optional\n"}, "/c")
	assert.True(t, on.TLSEnabled())
	assert.Equal(t, int64(0), on.Port(), "plaintext listener disabled")
	assert.Equal(t, "optional", on.TLSAuthClients())
}
