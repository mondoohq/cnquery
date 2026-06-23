// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStat(t *testing.T) {
	t.Run("ipv6 without opt field", func(t *testing.T) {
		lines := []string{
			"Chain OUTPUT (policy DROP 227 packets, 12904 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"2           0        0 ACCEPT     tcp      *      *       ::/0                 ::/0                 state NEW,ESTABLISHED",
		}
		expected := []Stat{
			{
				LineNumber:  2,
				Packets:     0,
				Bytes:       0,
				Target:      "ACCEPT",
				Protocol:    "tcp",
				Opt:         "  ",
				Input:       "*",
				Output:      "*",
				Source:      "::/0",
				Destination: "::/0",
				Options:     "state NEW,ESTABLISHED",
			},
		}
		result, err := ParseStat(lines, true)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("short/malformed lines are skipped without panicking", func(t *testing.T) {
		lines := []string{
			"Chain INPUT (policy ACCEPT 0 packets, 0 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"",                              // blank line in the middle of a block
			"1           0        0 ACCEPT", // truncated rule, fewer than 9 fields
			"2           0        0 ACCEPT     all  --  *      *       0.0.0.0/0            0.0.0.0/0",
		}
		var result []Stat
		var err error
		require.NotPanics(t, func() {
			result, err = ParseStat(lines, false)
		})
		require.NoError(t, err)
		// Only the well-formed rule survives; the blank and truncated lines are dropped.
		require.Len(t, result, 1)
		assert.Equal(t, int64(2), result[0].LineNumber)
		assert.Equal(t, "ACCEPT", result[0].Target)
		assert.Equal(t, "0.0.0.0/0", result[0].Source)
		assert.Equal(t, "0.0.0.0/0", result[0].Destination)
	})

	t.Run("ipv6 with opt field", func(t *testing.T) {
		lines := []string{
			"Chain OUTPUT (policy DROP 227 packets, 12904 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"2           0        0 ACCEPT     tcp    opt  *      *       ::/0                 ::/0                 state NEW,ESTABLISHED",
		}
		expected := []Stat{
			{
				LineNumber:  2,
				Packets:     0,
				Bytes:       0,
				Target:      "ACCEPT",
				Protocol:    "tcp",
				Opt:         "opt",
				Input:       "*",
				Output:      "*",
				Source:      "::/0",
				Destination: "::/0",
				Options:     "state NEW,ESTABLISHED",
			},
		}
		result, err := ParseStat(lines, true)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

func TestParseChainPolicy(t *testing.T) {
	t.Run("DROP policy with packet counts", func(t *testing.T) {
		lines := []string{
			"Chain INPUT (policy DROP 227 packets, 12904 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
		}
		assert.Equal(t, "DROP", ParseChainPolicy(lines))
	})

	t.Run("ACCEPT policy without packet counts", func(t *testing.T) {
		lines := []string{
			"Chain FORWARD (policy ACCEPT)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
		}
		assert.Equal(t, "ACCEPT", ParseChainPolicy(lines))
	})

	t.Run("REJECT policy", func(t *testing.T) {
		lines := []string{
			"Chain OUTPUT (policy REJECT 0 packets, 0 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
		}
		assert.Equal(t, "REJECT", ParseChainPolicy(lines))
	})

	t.Run("user-defined chain without policy", func(t *testing.T) {
		lines := []string{
			"Chain DOCKER (1 references)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
		}
		assert.Equal(t, "", ParseChainPolicy(lines))
	})

	t.Run("empty lines", func(t *testing.T) {
		assert.Equal(t, "", ParseChainPolicy([]string{}))
	})

	t.Run("nil lines", func(t *testing.T) {
		assert.Equal(t, "", ParseChainPolicy(nil))
	})
}

func TestParseChain(t *testing.T) {
	t.Run("full chain with policy and entries", func(t *testing.T) {
		lines := []string{
			"Chain INPUT (policy DROP 0 packets, 0 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"1         100      8000 ACCEPT     all  --  lo     *       0.0.0.0/0            0.0.0.0/0",
			"2          50      4000 DROP       all  --  *      *       127.0.0.0/8          0.0.0.0/0",
		}
		result, err := ParseChain(lines, false)
		require.NoError(t, err)
		assert.Equal(t, "DROP", result.Policy)
		require.Len(t, result.Entries, 2)
		assert.Equal(t, "ACCEPT", result.Entries[0].Target)
		assert.Equal(t, "lo", result.Entries[0].Input)
		assert.Equal(t, "DROP", result.Entries[1].Target)
		assert.Equal(t, "127.0.0.0/8", result.Entries[1].Source)
	})

	t.Run("empty chain with policy", func(t *testing.T) {
		lines := []string{
			"Chain FORWARD (policy DROP 0 packets, 0 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
		}
		result, err := ParseChain(lines, false)
		require.NoError(t, err)
		assert.Equal(t, "DROP", result.Policy)
		assert.Empty(t, result.Entries)
	})

	t.Run("ipv6 chain", func(t *testing.T) {
		lines := []string{
			"Chain INPUT (policy ACCEPT 0 packets, 0 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"1           0        0 ACCEPT     tcp      *      *       ::/0                 ::/0                 state NEW,ESTABLISHED",
		}
		result, err := ParseChain(lines, true)
		require.NoError(t, err)
		assert.Equal(t, "ACCEPT", result.Policy)
		require.Len(t, result.Entries, 1)
		assert.Equal(t, "::/0", result.Entries[0].Source)
	})
}

func TestSplitChainBlocks(t *testing.T) {
	t.Run("single chain", func(t *testing.T) {
		output := "Chain INPUT (policy ACCEPT 0 packets, 0 bytes)\nnum      pkts      bytes target     prot opt in     out     source               destination\n"
		blocks := splitChainBlocks(output)
		require.Len(t, blocks, 1)
		assert.Contains(t, blocks[0], "Chain INPUT")
	})

	t.Run("multiple chains with blank line separators", func(t *testing.T) {
		output := `Chain PREROUTING (policy ACCEPT 5 packets, 260 bytes)
num      pkts      bytes target     prot opt in     out     source               destination
1           5      260 DNAT       tcp  --  *      *       0.0.0.0/0            0.0.0.0/0            tcp dpt:80 to:192.168.1.100:8080

Chain INPUT (policy ACCEPT 0 packets, 0 bytes)
num      pkts      bytes target     prot opt in     out     source               destination

Chain OUTPUT (policy ACCEPT 3 packets, 148 bytes)
num      pkts      bytes target     prot opt in     out     source               destination

Chain POSTROUTING (policy ACCEPT 3 packets, 148 bytes)
num      pkts      bytes target     prot opt in     out     source               destination
1          10      600 MASQUERADE  all  --  *      eth0    192.168.1.0/24       0.0.0.0/0
`
		blocks := splitChainBlocks(output)
		require.Len(t, blocks, 4)
		assert.Contains(t, blocks[0], "Chain PREROUTING")
		assert.Contains(t, blocks[0], "DNAT")
		assert.Contains(t, blocks[1], "Chain INPUT")
		assert.Contains(t, blocks[2], "Chain OUTPUT")
		assert.Contains(t, blocks[3], "Chain POSTROUTING")
		assert.Contains(t, blocks[3], "MASQUERADE")
	})

	t.Run("empty output", func(t *testing.T) {
		blocks := splitChainBlocks("")
		assert.Empty(t, blocks)
	})
}

func TestParseChainName(t *testing.T) {
	assert.Equal(t, "INPUT", parseChainName("Chain INPUT (policy DROP 0 packets, 0 bytes)"))
	assert.Equal(t, "FORWARD", parseChainName("Chain FORWARD (policy ACCEPT)"))
	assert.Equal(t, "DOCKER", parseChainName("Chain DOCKER (1 references)"))
	assert.Equal(t, "PREROUTING", parseChainName("Chain PREROUTING (policy ACCEPT 5 packets, 260 bytes)"))
	assert.Equal(t, "POSTROUTING", parseChainName("Chain POSTROUTING (policy ACCEPT 0 packets, 0 bytes)"))
	assert.Equal(t, "", parseChainName(""))
	assert.Equal(t, "", parseChainName("not a chain header"))
}

func TestSplitChainBlocks_NATTable(t *testing.T) {
	output := `Chain PREROUTING (policy ACCEPT 0 packets, 0 bytes)
num      pkts      bytes target     prot opt in     out     source               destination
1           0        0 DNAT       tcp  --  eth0   *       0.0.0.0/0            0.0.0.0/0            tcp dpt:80 to:10.0.0.5:8080
2           0        0 DNAT       tcp  --  eth0   *       0.0.0.0/0            0.0.0.0/0            tcp dpt:443 to:10.0.0.5:8443

Chain INPUT (policy ACCEPT 0 packets, 0 bytes)
num      pkts      bytes target     prot opt in     out     source               destination

Chain OUTPUT (policy ACCEPT 0 packets, 0 bytes)
num      pkts      bytes target     prot opt in     out     source               destination

Chain POSTROUTING (policy ACCEPT 0 packets, 0 bytes)
num      pkts      bytes target     prot opt in     out     source               destination
1           0        0 MASQUERADE  all  --  *      eth0    10.0.0.0/24          0.0.0.0/0
2           0        0 SNAT       all  --  *      eth1    192.168.1.0/24       0.0.0.0/0            to:203.0.113.5
`

	blocks := splitChainBlocks(output)
	require.Len(t, blocks, 4)

	// Verify PREROUTING has DNAT rules
	preLines := getLines(blocks[0])
	result, err := ParseChain(preLines, false)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPT", result.Policy)
	require.Len(t, result.Entries, 2)
	assert.Equal(t, "DNAT", result.Entries[0].Target)
	assert.Equal(t, "eth0", result.Entries[0].Input)
	assert.Contains(t, result.Entries[0].Options, "tcp dpt:80 to:10.0.0.5:8080")
	assert.Equal(t, "DNAT", result.Entries[1].Target)
	assert.Contains(t, result.Entries[1].Options, "tcp dpt:443")

	// Verify POSTROUTING has MASQUERADE and SNAT
	postLines := getLines(blocks[3])
	result, err = ParseChain(postLines, false)
	require.NoError(t, err)
	assert.Equal(t, "ACCEPT", result.Policy)
	require.Len(t, result.Entries, 2)
	assert.Equal(t, "MASQUERADE", result.Entries[0].Target)
	assert.Equal(t, "10.0.0.0/24", result.Entries[0].Source)
	assert.Equal(t, "SNAT", result.Entries[1].Target)
	assert.Equal(t, "192.168.1.0/24", result.Entries[1].Source)
	assert.Contains(t, result.Entries[1].Options, "to:203.0.113.5")
}
