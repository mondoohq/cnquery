package os

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStat(t *testing.T) {
	// not opt
	t.Run("parsing conf lines", func(t *testing.T) {
		line := []string{"Chain OUTPUT (policy DROP 227 packets, 12904 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"2           0        0 ACCEPT     tcp      *      *       ::/0                 ::/0                 state NEW,ESTABLISHED"}
		expected := []Stat{{
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
			Options:     "state NEW,ESTABLISHED"},
		}
		result, err := ParseStat(line, true)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
	//opt
	t.Run("parsing conf lines", func(t *testing.T) {
		line := []string{"Chain OUTPUT (policy DROP 227 packets, 12904 bytes)",
			"num      pkts      bytes target     prot opt in     out     source               destination",
			"2           0        0 ACCEPT     tcp    opt  *      *       ::/0                 ::/0                 state NEW,ESTABLISHED"}
		expected := []Stat{{
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
			Options:     "state NEW,ESTABLISHED"},
		}
		result, err := ParseStat(line, true)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}
