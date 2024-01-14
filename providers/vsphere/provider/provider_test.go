// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/vsphere/connection/vsimulator"
)

func newTestService() (*vsimulator.VsphereSimulator, *Service, *plugin.ConnectRes) {
	vs, err := vsimulator.New()
	if err != nil {
		panic(err)
	}

	port, err := strconv.Atoi(vs.Server.URL.Port())
	if err != nil {
		panic(err)
	}

	srv := &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}

	resp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type:     "vsphere",
					Host:     vs.Server.URL.Hostname(),
					Port:     int32(port),
					Insecure: true, // allows self-signed certificates
					Credentials: []*vault.Credential{
						{
							Type:   vault.CredentialType_password,
							User:   vsimulator.Username,
							Secret: []byte(vsimulator.Password),
						},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		panic(err)
	}
	return vs, srv, resp
}

func TestResource_Vsphere(t *testing.T) {
	vs, srv, connRes := newTestService()
	defer vs.Close()

	// check that we get the data via the resources
	t.Run("simulate vsphere.datacenters[0].hosts[0].name", func(t *testing.T) {
		// simulate "vsphere.datacenters[0].hosts[0].name" where we expect "DC0_H0" as result

		// create vsphere resource
		dataResp, err := srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "vsphere",
		})
		if err != nil {
			panic(err)
		}
		resourceId := string(dataResp.Data.Value)

		// fetch datacenters
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "vsphere",
			ResourceId: resourceId,
			Field:      "datacenters",
		})
		if err != nil {
			panic(err)
		}

		// simulator has one datacenter /DC0
		assert.Equal(t, 1, len(dataResp.Data.Array))

		// get datacenter details
		datacenterResourceID := string(dataResp.Data.Array[0].Value)
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "vsphere.datacenter",
			ResourceId: datacenterResourceID,
			Field:      "name",
		})
		if err != nil {
			panic(err)
		}
		assert.Equal(t, "DC0", string(dataResp.Data.Value))

		// get list of hosts
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "vsphere.datacenter",
			ResourceId: datacenterResourceID,
			Field:      "hosts",
		})
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 4, len(dataResp.Data.Array))

		// we pick the first host on the first datacenter /DC0/host/DC0_H0/DC0_H0
		hostResourceID := string(dataResp.Data.Array[0].Value)
		dataResp, err = srv.GetData(&plugin.DataReq{
			Connection: connRes.Id,
			Resource:   "vsphere.host",
			ResourceId: hostResourceID,
			Field:      "name",
		})
		assert.Equal(t, "DC0_H0", string(dataResp.Data.Value))
	})
}

func TestVsphereDiscovery(t *testing.T) {
	vs, err := vsimulator.New()
	if err != nil {
		panic(err)
	}

	port, err := strconv.Atoi(vs.Server.URL.Port())
	if err != nil {
		panic(err)
	}

	srv := &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}

	resp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type:     "vsphere",
					Host:     vs.Server.URL.Hostname(),
					Port:     int32(port),
					Insecure: true, // allows self-signed certificates
					Discover: &inventory.Discovery{
						Targets: []string{"auto"},
					},
					Credentials: []*vault.Credential{
						{
							Type:   vault.CredentialType_password,
							User:   vsimulator.Username,
							Secret: []byte(vsimulator.Password),
						},
					},
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp.Asset)
	assert.Equal(t, 8, len(resp.Inventory.Spec.Assets)) // api + esx + vm
}
