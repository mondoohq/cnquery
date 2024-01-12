// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
)

type testPlugin struct {
	plugin.Service
}

func (t *testPlugin) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, nil
}

func (t *testPlugin) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, nil
}

func (t *testPlugin) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	return nil, nil
}

func (t *testPlugin) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	// sleep more than the heartbeat interval to ensure that even if shutting down
	// the provider can still respond to heartbeats
	time.Sleep(10 * time.Second)
	return &plugin.ShutdownRes{}, nil
}

func (t *testPlugin) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	return nil, nil
}

func (t *testPlugin) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	return nil, nil
}

func TestProviderShutdown(t *testing.T) {
	s := &RunningProvider{
		Plugin:      &testPlugin{},
		interval:    500 * time.Millisecond,
		gracePeriod: 500 * time.Millisecond,
	}
	err := s.heartbeat()
	require.NoError(t, err)
	require.False(t, s.isCloseOrShutdown())
	// the shutdown here takes 10 seconds, whereas the heartbeat interval is every second.
	// this means that this provider gets multiple heartbeats while shutting down
	err = s.Shutdown()
	require.NoError(t, err)
	require.True(t, s.isCloseOrShutdown())
}
