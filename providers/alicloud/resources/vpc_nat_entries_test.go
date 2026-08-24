// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	tea "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
)

func TestNatForwardsAllPorts(t *testing.T) {
	tests := []struct {
		name                       string
		externalPort, internalPort *string
		expected                   bool
	}{
		{"single port mapping", tea.String("443"), tea.String("8443"), false},
		// "any" on either end publishes every listening service on the private
		// address, not the one that was meant to be exposed
		{"any inbound", tea.String("any"), tea.String("any"), true},
		{"any only outside", tea.String("any"), tea.String("22"), true},
		{"any only inside", tea.String("22"), tea.String("any"), true},
		{"case insensitive", tea.String("Any"), tea.String("Any"), true},
		{"whitespace tolerated", tea.String(" any "), tea.String("443"), true},
		{"absent ports", nil, nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, natForwardsAllPorts(test.externalPort, test.internalPort))
		})
	}
}

func TestNatPortIsAny(t *testing.T) {
	assert.True(t, natPortIsAny(tea.String("any")))
	assert.True(t, natPortIsAny(tea.String("ANY")))
	assert.False(t, natPortIsAny(tea.String("443")))
	assert.False(t, natPortIsAny(tea.String("")))
	assert.False(t, natPortIsAny(nil))
}
