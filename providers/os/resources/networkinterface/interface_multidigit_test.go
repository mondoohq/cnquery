// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package networkinterface

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Interface indices routinely exceed 9 on real hosts (docker bridges,
// veth pairs, cloud VMs with many NICs). The parser must accept
// multi-digit indices instead of failing the whole enumeration.
func TestParseIpAddrMultiDigitIndex(t *testing.T) {
	input := "12: eth5    inet 10.0.0.5/24 brd 10.0.0.255 scope global eth5\\       valid_lft forever preferred_lft forever\n"

	h := &LinuxInterfaceHandler{}
	ifaces, err := h.ParseIpAddr(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, ifaces, 1)
	assert.Equal(t, "eth5", ifaces[0].Name)
	assert.Equal(t, 12, ifaces[0].Index)
}
