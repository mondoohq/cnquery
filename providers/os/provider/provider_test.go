package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/os/id/ids"
)

func TestLocalConnectionIdDetectors(t *testing.T) {
	// check that we get the data via the resources

	srv := &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}

	resp, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "local",
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Asset.IdDetector, 2)
	require.Contains(t, resp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, resp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, resp.Asset.IdDetector, ids.IdDetector_SshHostkey)
	// here we have the hostname twice, as platformid and stand alone
	// This get's cleaned up later in the code
	require.Len(t, resp.Asset.PlatformIds, 2)

	shutdownResp, err := srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownResp)

	srv = &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}
	resp, err = srv.Connect(&plugin.ConnectReq{
		Asset: resp.Asset,
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Asset.IdDetector, 2)
	require.Contains(t, resp.Asset.IdDetector, ids.IdDetector_Hostname)
	require.Contains(t, resp.Asset.IdDetector, ids.IdDetector_CloudDetect)
	require.NotContains(t, resp.Asset.IdDetector, ids.IdDetector_SshHostkey)
	// Now the platformIDs are cleaned up
	require.Len(t, resp.Asset.PlatformIds, 1)

	shutdownResp, err = srv.Shutdown(&plugin.ShutdownReq{})
	require.NoError(t, err)
	require.NotNil(t, shutdownResp)
}
