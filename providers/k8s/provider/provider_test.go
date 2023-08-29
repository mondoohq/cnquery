// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
)

func newTestService(t *testing.T, path string) (*Service, *plugin.ConnectRes) {
	srv := &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}

	resp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "k8s",
					Options: map[string]string{
						shared.OPTION_MANIFEST: path,
					},
				},
			},
		},
	}, nil)
	if err != nil {
		panic(err)
	}
	return srv, resp
}

func TestK8sServiceAccountAutomount(t *testing.T) {
	srv, connRes := newTestService(t, "../connection/shared/resources/testdata/serviceaccount-automount.yaml")

	dataResp, err := srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
	})
	require.NoError(t, err)
	resourceId := string(dataResp.Data.Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
		ResourceId: resourceId,
		Field:      "serviceaccounts",
	})
	require.NoError(t, err)

	// we have 1 service account
	assert.Equal(t, 1, len(dataResp.Data.Array))

	saResourceID := string(dataResp.Data.Array[0].Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s.serviceaccount",
		ResourceId: saResourceID,
		Field:      "automountServiceAccountToken",
	})
	require.NoError(t, err)

	assert.True(t, dataResp.Data.RawData().Value.(bool))
}

func TestK8sServiceAccountImplicitAutomount(t *testing.T) {
	srv, connRes := newTestService(t, "../connection/shared/resources/testdata/serviceaccount-implicit-automount.yaml")

	dataResp, err := srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
	})
	require.NoError(t, err)
	resourceId := string(dataResp.Data.Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
		ResourceId: resourceId,
		Field:      "serviceaccounts",
	})
	require.NoError(t, err)

	// we have 1 service account
	assert.Equal(t, 1, len(dataResp.Data.Array))

	saResourceID := string(dataResp.Data.Array[0].Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s.serviceaccount",
		ResourceId: saResourceID,
		Field:      "automountServiceAccountToken",
	})
	require.NoError(t, err)

	assert.True(t, dataResp.Data.RawData().Value.(bool))
}

func TestK8sServiceAccountNoAutomount(t *testing.T) {
	srv, connRes := newTestService(t, "../connection/shared/resources/testdata/serviceaccount-no-automount.yaml")

	dataResp, err := srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
	})
	require.NoError(t, err)
	resourceId := string(dataResp.Data.Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s",
		ResourceId: resourceId,
		Field:      "serviceaccounts",
	})
	require.NoError(t, err)

	// we have 1 service account
	assert.Equal(t, 1, len(dataResp.Data.Array))

	saResourceID := string(dataResp.Data.Array[0].Value)

	dataResp, err = srv.GetData(&plugin.DataReq{
		Connection: connRes.Id,
		Resource:   "k8s.serviceaccount",
		ResourceId: saResourceID,
		Field:      "automountServiceAccountToken",
	})
	require.NoError(t, err)

	assert.False(t, dataResp.Data.RawData().Value.(bool))
}

// TODO: this doesn't work now because a shared resource is created from the OS provider. The test
// panic in this case.
// func TestIngress(t *testing.T) {
// 	srv, connRes := newTestService(t, "../connection/shared/resources/testdata/ingress.yaml")

// 	dataResp, err := srv.GetData(&plugin.DataReq{
// 		Connection: connRes.Id,
// 		Resource:   "k8s",
// 	})
// 	require.NoError(t, err)
// 	resourceId := string(dataResp.Data.Value)

// 	dataResp, err = srv.GetData(&plugin.DataReq{
// 		Connection: connRes.Id,
// 		Resource:   "k8s",
// 		ResourceId: resourceId,
// 		Field:      "ingresses",
// 	})
// 	require.NoError(t, err)

// 	assert.Equal(t, 3, len(dataResp.Data.Array))

// 	t.Run("without-tls", func(t *testing.T) {
// 		tlsResp, err := srv.GetData(&plugin.DataReq{
// 			Connection: connRes.Id,
// 			Resource:   "k8s.ingress",
// 			ResourceId: string(dataResp.Data.Array[0].Value),
// 			Field:      "tls",
// 		})
// 		require.NoError(t, err)

// 		assert.Empty(t, tlsResp.Data.RawData().Value)
// 	})

// 	t.Run("with-tls", func(t *testing.T) {
// 		tlsResp, err := srv.GetData(&plugin.DataReq{
// 			Connection: connRes.Id,
// 			Resource:   "k8s.ingress",
// 			ResourceId: string(dataResp.Data.Array[1].Value),
// 			Field:      "tls",
// 		})
// 		require.NoError(t, err)

// 		assert.Empty(t, tlsResp.Data.RawData().Value)
// 	})

// 	t.Run("missing-tls-secret", func(t *testing.T) {
// 		tlsResp, err := srv.GetData(&plugin.DataReq{
// 			Connection: connRes.Id,
// 			Resource:   "k8s.ingress",
// 			ResourceId: string(dataResp.Data.Array[1].Value),
// 			Field:      "tls",
// 		})
// 		require.NoError(t, err)

// 		assert.Empty(t, tlsResp.Data.RawData().Value)
// 	})
// }
