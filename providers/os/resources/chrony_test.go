// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func chronySettings(lines ...string) []any {
	res := make([]any, len(lines))
	for i := range lines {
		res[i] = lines[i]
	}
	return res
}

func TestChronyDirectiveValues(t *testing.T) {
	settings := chronySettings(
		"server 0.pool.ntp.org iburst",
		"server 1.pool.ntp.org iburst",
		"pool 2.pool.ntp.org iburst maxsources 4",
		"allow 192.168.0.0/16",
		"deny all",
		"bindcmdaddress 127.0.0.1",
		"Server 3.pool.ntp.org", // case-insensitive directive match
	)

	require.Equal(t, []any{
		"0.pool.ntp.org iburst",
		"1.pool.ntp.org iburst",
		"3.pool.ntp.org",
	}, directiveValues(settings, "server"))

	require.Equal(t, []any{"2.pool.ntp.org iburst maxsources 4"}, directiveValues(settings, "pool"))
	require.Equal(t, []any{"192.168.0.0/16"}, directiveValues(settings, "allow"))
	require.Equal(t, []any{"all"}, directiveValues(settings, "deny"))
	require.Equal(t, []any{"127.0.0.1"}, directiveValues(settings, "bindcmdaddress"))
	require.Empty(t, directiveValues(settings, "peer"))
}

func TestChronyLastDirectiveValue(t *testing.T) {
	settings := chronySettings(
		"keyfile /etc/chrony.keys",
		"makestep 1.0 3",
		"keyfile /etc/chrony/override.keys", // last wins
	)

	require.Equal(t, "/etc/chrony/override.keys", lastDirectiveValue(settings, "keyfile"))
	require.Equal(t, "1.0 3", lastDirectiveValue(settings, "makestep"))
	require.Equal(t, "", lastDirectiveValue(settings, "leapsectz"))
}
