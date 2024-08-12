// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/os/id/ids"
)

func TestLocalConnectionIdDetectors(t *testing.T) {
	srv := &Service{
		Service: plugin.NewService(),
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

	require.Len(t, connectResp.Asset.IdDetector, 3)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_SerialNumber)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, connectResp.Asset.IdDetector, ids.IdDetector_SshHostkey)

	require.Len(t, connectResp.Asset.PlatformIds, 2)

	shutdownconnectResp, err := srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownconnectResp)

	srv = &Service{
		Service: plugin.NewService(),
	}
	connectResp, err = srv.Connect(&plugin.ConnectReq{
		Asset: connectResp.Asset,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	require.Len(t, connectResp.Asset.IdDetector, 3)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, connectResp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, connectResp.Asset.IdDetector, ids.IdDetector_SshHostkey)
	// Now the platformIDs are cleaned up
	require.Len(t, connectResp.Asset.PlatformIds, 2)

	shutdownconnectResp, err = srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownconnectResp)
}

func TestLocalConnectionIdDetectors_DelayedDiscovery(t *testing.T) {
	srv := &Service{
		Service: plugin.NewService(),
	}

	connectResp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type:           "docker-image",
					Host:           "alpine:3.19",
					DelayDiscovery: true,
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	require.Len(t, connectResp.Asset.IdDetector, 0)
	require.Len(t, connectResp.Asset.PlatformIds, 2)
	require.Nil(t, connectResp.Asset.Platform)

	// Disable delayed discovery and reconnect
	connectResp.Asset.Connections[0].DelayDiscovery = false
	connectResp, err = srv.Connect(&plugin.ConnectReq{
		Asset: connectResp.Asset,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	require.Len(t, connectResp.Asset.PlatformIds, 2)
	// Verify the platform is set
	require.NotNil(t, connectResp.Asset.Platform)

	shutdownconnectResp, err := srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownconnectResp)
}

func TestIdentifyDockerString(t *testing.T) {
	tests := []struct {
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

func TestService_ParseCLI(t *testing.T) {
	file, err := os.CreateTemp("/tmp", "cnquery_tests")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	s := &Service{
		Service: plugin.NewService(),
	}

	req := &plugin.ParseCLIReq{
		Connector: "ssh",
		Args:      []string{"pi@localhost"},
		Flags: map[string]*llx.Primitive{
			"password": {Value: []byte("password123")},
		},
	}
	res, err := s.ParseCLI(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Len(t, res.Asset.Connections[0].Credentials, 2)
	require.Equal(t, vault.CredentialType_password, res.Asset.Connections[0].Credentials[0].Type)
	require.Equal(t, vault.CredentialType_ssh_agent, res.Asset.Connections[0].Credentials[1].Type)

	req = &plugin.ParseCLIReq{
		Connector: "ssh",
		Args:      []string{"pi@localhost"},
	}
	res, err = s.ParseCLI(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Len(t, res.Asset.Connections[0].Credentials, 1)
	require.Equal(t, vault.CredentialType_ssh_agent, res.Asset.Connections[0].Credentials[0].Type)

	req = &plugin.ParseCLIReq{
		Connector: "ssh",
		Args:      []string{"pi@localhost"},
		Flags: map[string]*llx.Primitive{
			"identity-file": {Value: []byte(file.Name())},
		},
	}
	res, err = s.ParseCLI(req)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Len(t, res.Asset.Connections[0].Credentials, 1)
	require.Equal(t, vault.CredentialType_private_key, res.Asset.Connections[0].Credentials[0].Type)
}

func TestConnect_ContainerImage(t *testing.T) {
	srv := &Service{
		Service: plugin.NewService(),
	}

	connectResp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "docker-image",
					Host: "alpine:3.19.1",
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, connectResp)

	assert.Equal(t, "alpine", connectResp.Asset.Platform.Name)
	assert.Equal(t, "3.19.1", connectResp.Asset.Platform.Version)
	assert.Equal(t, "Alpine Linux v3.19", connectResp.Asset.Platform.Title)
	assert.Equal(t, "container-image", connectResp.Asset.Platform.Kind)
	assert.Equal(t, "docker-image", connectResp.Asset.Platform.Runtime)
}
