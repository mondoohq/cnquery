// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/network/connection"
	"go.mondoo.com/cnquery/v10/providers/network/resources"
	"go.mondoo.com/cnquery/v10/utils/syncx"
)

func TestResource_TLS(t *testing.T) {
	res := x.TestQuery(t, "tls(\"mondoo.com\").certificates")
	assert.NotEmpty(t, res)
	assert.Nil(t, res[0].Data.Error)
}

func TestResource_TlsFqdn(t *testing.T) {
	testCases := []struct {
		hostName   string
		expectedId string
	}{
		{
			hostName:   "www.mondoo.com",
			expectedId: "tls+tcp://www.mondoo.com:443",
		},
	}

	runtime := &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}

	for _, tc := range testCases {
		conf := &inventory.Config{
			Host: tc.hostName,
		}
		runtime.Connection = connection.NewHostConnection(1, &inventory.Asset{}, conf)

		dns, err := resources.NewResource(
			runtime,
			"tls",
			map[string]*llx.RawData{},
		)
		require.NoError(t, err)
		require.Equal(t, tc.expectedId, dns.MqlID())
	}
}
