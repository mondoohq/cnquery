// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package vcd

import (
	"fmt"
	"testing"

	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"

	"github.com/stretchr/testify/require"
)

func TestApiAccess(t *testing.T) {
	p, err := New(&providers.Config{
		Host: "<host>",
		Credentials: []*vault.Credential{
			{
				User:     "<user>",
				Password: "<password>",
			},
		},
		Options: map[string]string{
			"organization": "system",
		},
		Insecure: false,
	})
	require.NoError(t, err)

	client := p.Client()

	orgs, err := client.GetOrgList()
	require.NoError(t, err)
	for i := range orgs.Org {
		org := orgs.Org[i]
		fmt.Println(org.Name)
	}

	networks, err := client.GetExternalNetworks()
	require.NoError(t, err)
	for i := range networks.ExternalNetworkReference {
		net := networks.ExternalNetworkReference[i]
		fmt.Printf(net.Name)
	}
}
