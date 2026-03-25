// Copyright (c) Mondoo, Inc.
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
