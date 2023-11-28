// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/os/id/ids"
)

func TestLocalConnectionIdDetectors(t *testing.T) {
	srv := &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}

	connectResp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "local",
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	require.Len(t, connectResp.Asset.IdDetector, 2)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, connectResp.Asset.IdDetector, ids.IdDetector_SshHostkey)
	// here we have the hostname twice, as platformid and stand alone
	// This get's cleaned up later in the code
	// FIXME: this should only be 1
	require.Len(t, connectResp.Asset.PlatformIds, 2)

	shutdownconnectResp, err := srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownconnectResp)

	srv = &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}
	connectResp, err = srv.Connect(&plugin.ConnectReq{
		Asset: connectResp.Asset,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	require.Len(t, connectResp.Asset.IdDetector, 2)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, connectResp.Asset.IdDetector, ids.IdDetector_SshHostkey)
	// Now the platformIDs are cleaned up
	require.Len(t, connectResp.Asset.PlatformIds, 1)

	shutdownconnectResp, err = srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownconnectResp)
}

func TestIdentifyDockerString(t *testing.T) {
	var tests = []struct {
		input string
		want  string
	}{
		{"ubuntu:latest", "docker-image"},
		{"docker.io/pmuench/dvwa-container-escape", "docker-image"},
		{"registry.example.com:5000/myimage:latest", "docker-image"},
		{"4e2474c968d6", "docker-container"},
		{"my_container", "docker-container"},
		{"anotherContainer123", "docker-container"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s,%s", tt.input, tt.want), func(t *testing.T) {
			result := identifyContainerType(tt.input)
			assert.Equal(t, tt.want, result, "Mismatch for input: %s", tt.input)
		})
	}
}
