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

func TestResource_DNS(t *testing.T) {
	res := x.TestQuery(t, "dns(\"mondoo.com\").mx")
	assert.NotEmpty(t, res)
}

func TestResource_DomainName(t *testing.T) {
	res := x.TestQuery(t, "domainName")
	assert.NotEmpty(t, res)
	res = x.TestQuery(t, "domainName(\"mondoo.com\").tld")
	assert.Equal(t, "com", string(res[0].Result().Data.Value))
}

func TestResource_DnsFqdn(t *testing.T) {
	testCases := []struct {
		hostName   string
		expectedId string
	}{
		{
			hostName:   "127.0.0.1",
			expectedId: "dns/",
		},
		{
			hostName:   "3.127.139.132",
			expectedId: "dns/",
		},
		{
			hostName:   "www.mondoo.com",
			expectedId: "dns/www.mondoo.com",
		},
		{
			hostName:   "ec2-3-127-139-132.eu-central-1.compute.amazonaws.com",
			expectedId: "dns/ec2-3-127-139-132.eu-central-1.compute.amazonaws.com",
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
			"dns",
			map[string]*llx.RawData{},
		)
		require.NoError(t, err)
		require.Equal(t, tc.expectedId, dns.MqlID())
	}
}
