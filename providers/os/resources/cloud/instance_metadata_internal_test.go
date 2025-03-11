// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeIPs(t *testing.T) {
	ip1 := Ipv4Address{IP: "192.168.1.1", Subnet: "255.255.255.0"}
	ip2 := Ipv4Address{IP: "192.168.1.1", Gateway: "192.168.1.254"}
	merged := mergeIPs(ip1, ip2)

	assert.Equal(t, "192.168.1.1", merged.IP)
	assert.Equal(t, "255.255.255.0", merged.Subnet)
	assert.Equal(t, "192.168.1.254", merged.Gateway)
}
