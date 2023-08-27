// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package connection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

func TestGcpDiscovery(t *testing.T) {
	orgId := "<insert org id>"
	conf := &inventory.Config{
		Type: "gcp",
		Options: map[string]string{
			"organization": orgId,
		},
		Discover: &inventory.Discovery{
			Targets: []string{"all"},
		},
	}

	conn, err := NewGcpConnection(0, nil, conf)
	require.NoError(t, err)
	_, err = conn.GetOrganization(orgId)
	require.NoError(t, err)
}
