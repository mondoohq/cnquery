// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMlagInterfaces(t *testing.T) {
	f, err := os.ReadFile("./testdata/mlag-config")
	require.NoError(t, err)

	interfaces := ParseMlagInterfaces(string(f))

	// Should find 3 MLAG interfaces (Port-Channel1, Port-Channel2, Port-Channel10)
	// Port-Channel1000 should NOT be included (no mlag ID)
	assert.Len(t, interfaces, 3)

	// Verify the parsed interfaces
	expected := map[string]string{
		"Port-Channel1":  "1",
		"Port-Channel2":  "2",
		"Port-Channel10": "10",
	}

	for _, intf := range interfaces {
		expectedID, ok := expected[intf.Name]
		assert.True(t, ok, "unexpected interface: %s", intf.Name)
		assert.Equal(t, expectedID, intf.MlagID, "wrong MLAG ID for %s", intf.Name)
	}
}

func TestParseMlagInterfacesEmpty(t *testing.T) {
	// Config with no MLAG interfaces
	config := `! Command: show running-config
hostname switch1
!
interface Ethernet1
   no switchport
   ip address 10.0.0.1/24
!
end
`
	interfaces := ParseMlagInterfaces(config)
	assert.Len(t, interfaces, 0)
}

func TestParseMlagInterfacesNoMlagID(t *testing.T) {
	// Port-Channel without mlag ID should not be included
	config := `! Command: show running-config
!
interface Port-Channel1
   description No MLAG ID
   switchport mode trunk
!
interface Port-Channel2
   description Has MLAG ID
   switchport mode trunk
   mlag 5
!
end
`
	interfaces := ParseMlagInterfaces(config)
	assert.Len(t, interfaces, 1)
	assert.Equal(t, "Port-Channel2", interfaces[0].Name)
	assert.Equal(t, "5", interfaces[0].MlagID)
}

func TestParseMlagInterfacesMultiDigitID(t *testing.T) {
	// Test multi-digit MLAG IDs and Port-Channel numbers
	config := `!
interface Port-Channel100
   mlag 999
!
interface Port-Channel1234
   description High numbered
   mlag 1
!
end
`
	interfaces := ParseMlagInterfaces(config)
	assert.Len(t, interfaces, 2)

	// Build a map for easier verification
	intfMap := make(map[string]string)
	for _, intf := range interfaces {
		intfMap[intf.Name] = intf.MlagID
	}

	assert.Equal(t, "999", intfMap["Port-Channel100"])
	assert.Equal(t, "1", intfMap["Port-Channel1234"])
}
