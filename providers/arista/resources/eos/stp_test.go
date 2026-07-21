// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Locks in the fix for the hardcoded STP interface command. Previously GetCmd
// always returned `show spanning-tree mst 0 interface Ethernet1 detail`,
// ignoring the instance/interface arguments, so every StpInterfaceDetails call
// returned details for that single interface.
func TestStpInterfaceDetailCmd(t *testing.T) {
	t.Run("uses the provided instance and interface", func(t *testing.T) {
		cmd, err := stpInterfaceDetailCmd("10", "Ethernet5")
		require.NoError(t, err)
		assert.Equal(t, "show spanning-tree mst 10 interface Ethernet5 detail", cmd)
	})

	t.Run("default instance and interface", func(t *testing.T) {
		cmd, err := stpInterfaceDetailCmd("0", "Ethernet1")
		require.NoError(t, err)
		assert.Equal(t, "show spanning-tree mst 0 interface Ethernet1 detail", cmd)
	})

	t.Run("accepts real EOS interface name shapes", func(t *testing.T) {
		for _, iface := range []string{"Ethernet1/1", "Port-Channel3", "Ethernet1.100", "Vlan100", "Management1"} {
			cmd, err := stpInterfaceDetailCmd("2", iface)
			require.NoError(t, err, iface)
			assert.Equal(t, "show spanning-tree mst 2 interface "+iface+" detail", cmd)
		}
	})

	t.Run("GetCmd returns the command set on the request", func(t *testing.T) {
		cmd, err := stpInterfaceDetailCmd("2", "Port-Channel3")
		require.NoError(t, err)
		shRsp := &showSpanningTreeMstInstanceDetail{cmd: cmd}
		assert.Equal(t, "show spanning-tree mst 2 interface Port-Channel3 detail", shRsp.GetCmd())
	})

	t.Run("rejects malformed arguments", func(t *testing.T) {
		bad := []struct{ id, iface string }{
			{"0; reload", "Ethernet1"},       // injected instance id
			{"0", "Ethernet1 detail | grep"}, // space + metacharacters in iface
			{"abc", "Ethernet1"},             // non-numeric instance id
			{"0", ""},                        // empty iface
			{"", "Ethernet1"},                // empty instance id
		}
		for _, b := range bad {
			_, err := stpInterfaceDetailCmd(b.id, b.iface)
			assert.Error(t, err, "expected error for id=%q iface=%q", b.id, b.iface)
		}
	})
}
