// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"
)

func TestParseInfo(t *testing.T) {
	info := "# Server\r\nredis_version:7.4.0\r\nredis_mode:standalone\r\n\r\n# Clients\r\nconnected_clients:1\r\n"
	got := parseInfo(info)
	if got["redis_version"] != "7.4.0" {
		t.Errorf("redis_version = %q, want 7.4.0", got["redis_version"])
	}
	if got["redis_mode"] != "standalone" {
		t.Errorf("redis_mode = %q, want standalone", got["redis_mode"])
	}
	if _, ok := got["# Server"]; ok {
		t.Error("section headers should be skipped")
	}
}

func TestBindsAll(t *testing.T) {
	cases := []struct {
		bind []string
		want bool
	}{
		{nil, true},                            // no bind configured => all interfaces
		{[]string{}, true},                     // empty => all interfaces
		{[]string{"127.0.0.1", "-::1"}, false}, // loopback only
		{[]string{"0.0.0.0"}, true},            // wildcard
		{[]string{"127.0.0.1", "0.0.0.0"}, true},
		{[]string{"*"}, true},
		{[]string{"10.0.0.5"}, false},
	}
	for _, c := range cases {
		if got := bindsAll(c.bind); got != c.want {
			t.Errorf("bindsAll(%v) = %v, want %v", c.bind, got, c.want)
		}
	}
}

func TestIsNoPerm(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("NOPERM this user has no permissions to run the 'config|get' command"), true},
		{errors.New("WRONGPASS invalid username-password pair"), true},
		{errors.New("connection refused"), false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isNoPerm(c.err); got != c.want {
			t.Errorf("isNoPerm(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestParseACLLine(t *testing.T) {
	// default user: enabled, has a password hash, all keys/channels, all commands.
	def := parseACLLine("user default on #9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08 ~* &* +@all")
	if def.name != "default" || !def.enabled || def.nopass {
		t.Errorf("default parse: %+v", def)
	}
	if def.passwordCount != 1 {
		t.Errorf("default passwordCount = %d, want 1", def.passwordCount)
	}
	if len(def.keyPatterns) != 1 || def.keyPatterns[0] != "*" {
		t.Errorf("default keyPatterns = %v, want [*]", def.keyPatterns)
	}
	if len(def.commandRules) != 1 || def.commandRules[0] != "+@all" {
		t.Errorf("default commandRules = %v, want [+@all]", def.commandRules)
	}

	// restricted auditor: off, nopass, scoped key pattern, category rules.
	aud := parseACLLine("user auditor off nopass ~app:* &notifications:* -@all +@read +@connection")
	if aud.name != "auditor" || aud.enabled || !aud.nopass {
		t.Errorf("auditor parse: %+v", aud)
	}
	if aud.passwordCount != 0 {
		t.Errorf("auditor passwordCount = %d, want 0", aud.passwordCount)
	}
	if len(aud.keyPatterns) != 1 || aud.keyPatterns[0] != "app:*" {
		t.Errorf("auditor keyPatterns = %v, want [app:*]", aud.keyPatterns)
	}
	if len(aud.channelPatterns) != 1 || aud.channelPatterns[0] != "notifications:*" {
		t.Errorf("auditor channelPatterns = %v, want [notifications:*]", aud.channelPatterns)
	}
	want := []string{"-@all", "+@read", "+@connection"}
	if len(aud.commandRules) != len(want) {
		t.Fatalf("auditor commandRules = %v, want %v", aud.commandRules, want)
	}
	for i, w := range want {
		if aud.commandRules[i] != w {
			t.Errorf("commandRules[%d] = %v, want %s", i, aud.commandRules[i], w)
		}
	}

	// allkeys/allchannels keywords normalize to "*".
	ak := parseACLLine("user wide on nopass allkeys allchannels +@all")
	if len(ak.keyPatterns) != 1 || ak.keyPatterns[0] != "*" {
		t.Errorf("allkeys => %v, want [*]", ak.keyPatterns)
	}
	if len(ak.channelPatterns) != 1 || ak.channelPatterns[0] != "*" {
		t.Errorf("allchannels => %v, want [*]", ak.channelPatterns)
	}

	// a non-user line yields an empty name and is skipped by the caller.
	if got := parseACLLine("not an acl line"); got.name != "" {
		t.Errorf("non-user line name = %q, want empty", got.name)
	}
}

func TestAtoiOr(t *testing.T) {
	if got := atoiOr("6379", 0); got != 6379 {
		t.Errorf("atoiOr(6379) = %d", got)
	}
	if got := atoiOr("", 42); got != 42 {
		t.Errorf("atoiOr(empty) = %d, want 42", got)
	}
	if got := atoiOr("notanumber", 7); got != 7 {
		t.Errorf("atoiOr(bad) = %d, want 7", got)
	}
}
