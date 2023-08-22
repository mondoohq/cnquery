// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package dnsshake

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDnsShake(t *testing.T) {
	dnsShaker, err := New("mondoo.io")
	require.NoError(t, err)

	records, err := dnsShaker.Query("A", "MX")
	require.NoError(t, err)
	assert.True(t, len(records) > 0)
}
