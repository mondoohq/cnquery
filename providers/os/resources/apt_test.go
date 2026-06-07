// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAptOneLine(t *testing.T) {
	content := `# Ubuntu sources
deb http://archive.ubuntu.com/ubuntu noble main restricted universe
deb-src http://archive.ubuntu.com/ubuntu noble main
deb [trusted=yes signed-by=/usr/share/keyrings/foo.gpg] https://repo.example.com/apt stable main
# deb http://disabled.example.com/ubuntu noble main
deb [arch=amd64] http://only-options.example.com noble main

not a repo line
deb http://incomplete.example.com`

	repos := parseAptOneLine(content)
	require.Len(t, repos, 5)

	require.Equal(t, aptRepo{
		Type:         "deb",
		URL:          "http://archive.ubuntu.com/ubuntu",
		Distribution: "noble",
		Components:   []string{"main", "restricted", "universe"},
		Enabled:      true,
	}, repos[0])

	require.Equal(t, aptRepo{
		Type:         "deb-src",
		URL:          "http://archive.ubuntu.com/ubuntu",
		Distribution: "noble",
		Components:   []string{"main"},
		Enabled:      true,
	}, repos[1])

	// trusted + signed-by options are parsed
	require.Equal(t, "https://repo.example.com/apt", repos[2].URL)
	require.True(t, repos[2].Trusted)
	require.Equal(t, "/usr/share/keyrings/foo.gpg", repos[2].SignedBy)
	require.Equal(t, []string{"main"}, repos[2].Components)

	// commented-out repo is captured as disabled
	require.Equal(t, "http://disabled.example.com/ubuntu", repos[3].URL)
	require.False(t, repos[3].Enabled)

	// unrelated options leave trusted/signedBy untouched
	require.Equal(t, "http://only-options.example.com", repos[4].URL)
	require.False(t, repos[4].Trusted)
	require.Empty(t, repos[4].SignedBy)
}

func TestParseAptDeb822(t *testing.T) {
	content := `# managed by some tool
Types: deb deb-src
URIs: http://archive.ubuntu.com/ubuntu
Suites: noble noble-updates
Components: main restricted
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: https://untrusted.example.com/apt
Suites: stable
Components: main
Trusted: yes
Enabled: no`

	repos := parseAptDeb822(content)

	// first stanza: 2 types x 1 uri x 2 suites = 4 repos
	// second stanza: 1 repo
	require.Len(t, repos, 5)

	require.Equal(t, "deb", repos[0].Type)
	require.Equal(t, "http://archive.ubuntu.com/ubuntu", repos[0].URL)
	require.Equal(t, "noble", repos[0].Distribution)
	require.Equal(t, []string{"main", "restricted"}, repos[0].Components)
	require.Equal(t, "/usr/share/keyrings/ubuntu-archive-keyring.gpg", repos[0].SignedBy)
	require.True(t, repos[0].Enabled)
	require.False(t, repos[0].Trusted)

	require.Equal(t, "noble-updates", repos[1].Distribution)
	require.Equal(t, "deb-src", repos[2].Type)

	// second stanza: trusted + disabled
	last := repos[4]
	require.Equal(t, "https://untrusted.example.com/apt", last.URL)
	require.True(t, last.Trusted)
	require.False(t, last.Enabled)
}

func TestAptBool(t *testing.T) {
	for _, v := range []string{"yes", "YES", "true", "1", " yes "} {
		require.True(t, aptBool(v), v)
	}
	for _, v := range []string{"no", "false", "0", "", "maybe"} {
		require.False(t, aptBool(v), v)
	}
}
