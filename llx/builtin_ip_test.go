// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDottedMask(t *testing.T) {
	tests := []struct {
		ip       string
		mask     string
		expected RawIP
	}{
		{
			ip:   "192.168.1.1",
			mask: "",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    24, // default mask
				HasPrefixLength: false,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "1.2.3",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    24, // default mask
				HasPrefixLength: false,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.0.255.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    24, // default mask
				HasPrefixLength: false,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "350.450.555.678",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    24, // default mask
				HasPrefixLength: false,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "0.0.0.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    0,
				HasPrefixLength: true,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.0.0.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				Version:         4,
				PrefixLength:    8,
				HasPrefixLength: true,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.255.0.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				PrefixLength:    16,
				HasPrefixLength: true,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.255.255.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				PrefixLength:    24,
				HasPrefixLength: true,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.255.255.255",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				PrefixLength:    32,
				HasPrefixLength: true,
			},
		},
		{
			ip:   "192.168.1.1",
			mask: "255.255.240.0",
			expected: RawIP{
				IP:              net.ParseIP("192.168.1.1"),
				PrefixLength:    20,
				HasPrefixLength: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.ip, tt.mask), func(t *testing.T) {
			result := ParseIPv4WithDottedMask(tt.ip, tt.mask)
			assert.Equal(t, tt.expected.IP, result.IP)
			assert.Equal(t, uint8(4), result.Version)
			assert.Equal(t, tt.expected.PrefixLength, result.PrefixLength)
			assert.Equal(t, tt.expected.HasPrefixLength, result.HasPrefixLength)
		})
	}
}

func TestCreateMask(t *testing.T) {
	tests := []struct {
		mask     int
		offset   int
		maxBytes int
		res      []byte
	}{
		{0, 0, 1, []byte{0x00}},
		{1, 0, 1, []byte{0x80}},
		{5, 0, 1, []byte{0xf8}},
		{8, 0, 1, []byte{0xff}},
		{9, 0, 1, []byte{0xff}},
		{9, 0, 2, []byte{0xff, 0x80}},
		{4, 4, 1, []byte{0x0f}},
		{7, 1, 1, []byte{0x7f}},
		{5, 3, 1, []byte{0x1f}},
		{6, 3, 2, []byte{0x1f, 0x80}},
		{6, 3, 1, []byte{0x1f}},
		{16, 48, 16, []byte{0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0}},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(fmt.Sprintf("bits=%d off=%d max=%d", cur.mask, cur.offset, cur.maxBytes), func(t *testing.T) {
			res := createMask(cur.mask, cur.offset, cur.maxBytes)
			assert.Equal(t, cur.res, res)
		})
	}
}

func TestIntIP(t *testing.T) {
	assert.Equal(t, net.ParseIP("172.0.0.1"), int2ip(2885681153))
	assert.Equal(t, net.ParseIP("0.0.0.0"), int2ip(0))
	assert.Equal(t, net.ParseIP(""), net.IP(nil))
	assert.Equal(t, net.ParseIP("255.255.255.255"), int2ip(1<<33-1))
}
