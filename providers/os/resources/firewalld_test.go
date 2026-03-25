// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleFirewalldOutput = `block
  target: %%REJECT%%
  icmp-block-inversion: no
  interfaces:
  sources:
  services:
  ports:
  protocols:
  forward: yes
  masquerade: no
  forward-ports:
  source-ports:
  icmp-blocks:
  rich rules:

dmz
  target: default
  icmp-block-inversion: no
  interfaces:
  sources:
  services: ssh
  ports:
  protocols:
  forward: yes
  masquerade: no
  forward-ports:
  source-ports:
  icmp-blocks:
  rich rules:

public (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0 eth1
  sources:
  services: cockpit dhcpv6-client ssh
  ports: 8080/tcp 443/tcp
  protocols: icmp
  forward: yes
  masquerade: yes
  forward-ports: port=80:proto=tcp:toport=8080:toaddr=
  source-ports:
  icmp-blocks: echo-reply
  rich rules:
	rule family="ipv4" source address="10.0.0.0/8" accept
	rule family="ipv4" source address="192.168.1.0/24" destination address="10.0.0.0/8" reject
`

func TestParseFirewalldZones(t *testing.T) {
	zones := parseFirewalldZones(sampleFirewalldOutput)
	require.Len(t, zones, 3)

	// block zone
	block := zones[0]
	assert.Equal(t, "block", block.name)
	assert.Equal(t, "%%REJECT%%", block.target)
	assert.False(t, block.active)
	assert.False(t, block.icmpBlockInversion)
	assert.Nil(t, block.interfaces)
	assert.Nil(t, block.services)
	assert.False(t, block.masquerade)

	// dmz zone
	dmz := zones[1]
	assert.Equal(t, "dmz", dmz.name)
	assert.Equal(t, "default", dmz.target)
	assert.False(t, dmz.active)
	assert.Equal(t, []string{"ssh"}, dmz.services)

	// public zone (active)
	pub := zones[2]
	assert.Equal(t, "public", pub.name)
	assert.True(t, pub.active)
	assert.Equal(t, "default", pub.target)
	assert.Equal(t, []string{"eth0", "eth1"}, pub.interfaces)
	assert.Equal(t, []string{"cockpit", "dhcpv6-client", "ssh"}, pub.services)
	assert.Equal(t, []string{"8080/tcp", "443/tcp"}, pub.ports)
	assert.Equal(t, []string{"icmp"}, pub.protocols)
	assert.True(t, pub.masquerade)
	assert.Equal(t, []string{"port=80:proto=tcp:toport=8080:toaddr="}, pub.forwardPorts)
	assert.Equal(t, []string{"echo-reply"}, pub.icmpBlocks)
	require.Len(t, pub.richRules, 2)
	assert.Equal(t, `rule family="ipv4" source address="10.0.0.0/8" accept`, pub.richRules[0])
	assert.Equal(t, `rule family="ipv4" source address="192.168.1.0/24" destination address="10.0.0.0/8" reject`, pub.richRules[1])
}

func TestParseFirewalldZonesEmpty(t *testing.T) {
	zones := parseFirewalldZones("")
	assert.Empty(t, zones)
}

func TestParseFirewalldRichRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected parsedRichRule
	}{
		{
			name:  "accept with source",
			input: `rule family="ipv4" source address="10.0.0.0/8" accept`,
			expected: parsedRichRule{
				family: "ipv4",
				source: "10.0.0.0/8",
				action: "accept",
			},
		},
		{
			name:  "reject with source and destination",
			input: `rule family="ipv4" source address="192.168.1.0/24" destination address="10.0.0.0/8" reject`,
			expected: parsedRichRule{
				family:      "ipv4",
				source:      "192.168.1.0/24",
				destination: "10.0.0.0/8",
				action:      "reject",
			},
		},
		{
			name:  "ipv6 drop",
			input: `rule family="ipv6" source address="fd00::/8" drop`,
			expected: parsedRichRule{
				family: "ipv6",
				source: "fd00::/8",
				action: "drop",
			},
		},
		{
			name:  "no family",
			input: `rule source address="10.0.0.0/8" accept`,
			expected: parsedRichRule{
				source: "10.0.0.0/8",
				action: "accept",
			},
		},
		{
			name:  "service name containing action keyword",
			input: `rule family="ipv4" service name="acceptor" drop`,
			expected: parsedRichRule{
				family: "ipv4",
				action: "drop",
			},
		},
		{
			name:  "log with accept",
			input: `rule family="ipv4" source address="10.0.0.0/8" log prefix="test" level="info" accept`,
			expected: parsedRichRule{
				family: "ipv4",
				source: "10.0.0.0/8",
				action: "accept",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFirewalldRichRule(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitNonEmpty(t *testing.T) {
	assert.Nil(t, splitNonEmpty(""))
	assert.Equal(t, []string{"ssh"}, splitNonEmpty("ssh"))
	assert.Equal(t, []string{"ssh", "http", "https"}, splitNonEmpty("ssh http https"))
	assert.Equal(t, []string{"a", "b"}, splitNonEmpty("  a   b  "))
}
