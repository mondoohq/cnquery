// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"00:11:22:33:44:55", "00:11:22:33:44:55"},
		{"AA-BB-CC-DD-EE-FF", "aa:bb:cc:dd:ee:ff"}, // windows dashes
		{"a:b:c:d:e:f", "0a:0b:0c:0d:0e:0f"},       // macOS unpadded octets
		{"(incomplete)", ""},                       // macOS incomplete entry
		{"00:00:00:00:00:00", ""},                  // all-zero
		{"", ""},                                   // empty
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, normalizeMAC(tt.in), "normalizeMAC(%q)", tt.in)
	}
}

func TestParseIPNeighJSON(t *testing.T) {
	out := `[
		{"dst":"192.168.1.1","dev":"eth0","lladdr":"00:11:22:33:44:55","state":["REACHABLE"]},
		{"dst":"192.168.1.50","dev":"eth0","lladdr":"aa:bb:cc:dd:ee:ff","state":["STALE"]},
		{"dst":"192.168.1.99","dev":"eth0","state":["FAILED"]},
		{"dst":"fe80::1","dev":"eth0","lladdr":"00:11:22:33:44:66","state":["REACHABLE"]}
	]`
	got, err := parseIPNeighJSON(out)
	require.NoError(t, err)
	require.Len(t, got, 4)

	require.Equal(t, Neighbor{IP: "192.168.1.1", MAC: "00:11:22:33:44:55", Interface: "eth0", State: "reachable"}, got[0])
	require.Equal(t, "stale", got[1].State)
	// FAILED entry has no link-layer address.
	require.Equal(t, Neighbor{IP: "192.168.1.99", MAC: "", Interface: "eth0", State: "failed"}, got[2])
	require.Equal(t, "fe80::1", got[3].IP)
}

func TestParseProcNetArp(t *testing.T) {
	content := "IP address       HW type     Flags       HW address            Mask     Device\n" +
		"192.168.1.1      0x1         0x2         00:11:22:33:44:55     *        eth0\n" +
		"192.168.1.99     0x1         0x0         00:00:00:00:00:00     *        eth0\n"
	got := parseProcNetArp(content)
	require.Len(t, got, 2)
	require.Equal(t, Neighbor{IP: "192.168.1.1", MAC: "00:11:22:33:44:55", Interface: "eth0", State: "reachable"}, got[0])
	// Flags 0x0 → incomplete, all-zero MAC normalizes to empty.
	require.Equal(t, Neighbor{IP: "192.168.1.99", MAC: "", Interface: "eth0", State: "incomplete"}, got[1])
}

func TestParseArpAn(t *testing.T) {
	out := "? (192.168.1.1) at 0:11:22:33:44:55 on en0 ifscope [ethernet]\n" +
		"? (192.168.1.5) at (incomplete) on en0 ifscope [ethernet]\n" +
		"foo.local (10.0.0.2) at a:b:c:d:e:f on en1 [ethernet]\n" +
		"? (224.0.0.251) at 1:0:5e:0:0:fb on en0 ifscope permanent [ethernet]\n"
	got := parseArpAn(out)
	require.Len(t, got, 4)
	require.Equal(t, Neighbor{IP: "192.168.1.1", MAC: "00:11:22:33:44:55", Interface: "en0", State: "reachable"}, got[0])
	require.Equal(t, Neighbor{IP: "192.168.1.5", MAC: "", Interface: "en0", State: "incomplete"}, got[1])
	require.Equal(t, Neighbor{IP: "10.0.0.2", MAC: "0a:0b:0c:0d:0e:0f", Interface: "en1", State: "reachable"}, got[2])
	// `permanent` (static) entries must not be flattened to "reachable".
	require.Equal(t, Neighbor{IP: "224.0.0.251", MAC: "01:00:5e:00:00:fb", Interface: "en0", State: "permanent"}, got[3])
}

func TestParseNdpAn(t *testing.T) {
	out := "Neighbor                             Linklayer Address  Netif Expire    St Flgs Prbs\n" +
		"fe80::1%en0                          d4:24:dd:d0:b9:5e   en0 permanent  R  R\n" +
		"2001:db8::5                          aa:bb:cc:dd:ee:ff   en0 23h59m58s  S\n" +
		"fe80::abc%en0                        (incomplete)        en0 expired    I\n"
	got := parseNdpAn(out)
	require.Len(t, got, 3)
	// %zone stripped; permanent (Expire col) wins over the St code.
	require.Equal(t, Neighbor{IP: "fe80::1", MAC: "d4:24:dd:d0:b9:5e", Interface: "en0", State: "permanent"}, got[0])
	require.Equal(t, Neighbor{IP: "2001:db8::5", MAC: "aa:bb:cc:dd:ee:ff", Interface: "en0", State: "stale"}, got[1])
	require.Equal(t, Neighbor{IP: "fe80::abc", MAC: "", Interface: "en0", State: "incomplete"}, got[2])
}

func TestParseWindowsNeighbors(t *testing.T) {
	// Array result.
	out := `[
		{"Ip":"192.168.1.1","Mac":"00-11-22-33-44-55","Interface":"Ethernet","State":"Reachable"},
		{"Ip":"192.168.1.2","Mac":"AA-BB-CC-DD-EE-FF","Interface":"Ethernet","State":"Stale"}
	]`
	got, err := parseWindowsNeighbors(out)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, Neighbor{IP: "192.168.1.1", MAC: "00:11:22:33:44:55", Interface: "Ethernet", State: "reachable"}, got[0])
	require.Equal(t, Neighbor{IP: "192.168.1.2", MAC: "aa:bb:cc:dd:ee:ff", Interface: "Ethernet", State: "stale"}, got[1])

	// PowerShell emits a bare object for a single result.
	single, err := parseWindowsNeighbors(`{"Ip":"10.0.0.1","Mac":"00-11-22-33-44-55","Interface":"Ethernet 2","State":"Permanent"}`)
	require.NoError(t, err)
	require.Len(t, single, 1)
	require.Equal(t, "permanent", single[0].State)

	// Empty output.
	empty, err := parseWindowsNeighbors("")
	require.NoError(t, err)
	require.Empty(t, empty)
}
