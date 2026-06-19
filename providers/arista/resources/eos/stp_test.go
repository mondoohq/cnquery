// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Locks in the fix for the hardcoded STP interface command. Previously GetCmd
// always returned `show spanning-tree mst 0 interface Ethernet1 detail`,
// ignoring the instance/interface arguments, so every StpInterfaceDetails call
// returned details for that single interface.
func TestStpInterfaceDetailCmd(t *testing.T) {
	t.Run("uses the provided instance and interface", func(t *testing.T) {
		assert.Equal(t,
			"show spanning-tree mst 10 interface Ethernet5 detail",
			stpInterfaceDetailCmd("10", "Ethernet5"))
	})

	t.Run("default instance and interface", func(t *testing.T) {
		assert.Equal(t,
			"show spanning-tree mst 0 interface Ethernet1 detail",
			stpInterfaceDetailCmd("0", "Ethernet1"))
	})

	t.Run("GetCmd returns the command set on the request", func(t *testing.T) {
		shRsp := &showSpanningTreeMstInstanceDetail{cmd: stpInterfaceDetailCmd("2", "Port-Channel3")}
		assert.Equal(t, "show spanning-tree mst 2 interface Port-Channel3 detail", shRsp.GetCmd())
	})
}
