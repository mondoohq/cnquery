// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"bytes"
	"unicode/utf8"

	plugin "github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	var x ProviderPlugin = &GRPCClient{}
	_ = x
}

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCClient struct {
	broker *plugin.GRPCBroker
	client ProviderPluginClient
}

func (m *GRPCClient) Heartbeat(req *HeartbeatReq) (*HeartbeatRes, error) {
	return m.client.Heartbeat(context.Background(), req)
}

func (m *GRPCClient) ParseCLI(req *ParseCLIReq) (*ParseCLIRes, error) {
	return m.client.ParseCLI(context.Background(), req)
}

func (m *GRPCClient) connect(req *ConnectReq, callback ProviderCallback) {
	helper := &GRPCProviderCallbackServer{Impl: callback}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		RegisterProviderCallbackServer(s, helper)
		return s
	}

	brokerID := m.broker.NextId()
	req.CallbackServer = brokerID
	go m.broker.AcceptAndServe(brokerID, serverFunc)

	// Note: the reverse connection is not closed explicitly. It stays open
	// until the process is eventually stopped. Connect should only be called
	// once per connected asset, thus the reverse connection is also only
	// open for the duration of said connection.
	// In the future, we may want to explicitly disconnect and re-use providers.
}

func (m *GRPCClient) Connect(req *ConnectReq, callback ProviderCallback) (*ConnectRes, error) {
	m.connect(req, callback)
	return m.client.Connect(context.Background(), req)
}

func (m *GRPCClient) Disconnect(req *DisconnectReq) (*DisconnectRes, error) {
	return m.client.Disconnect(context.Background(), req)
}

func (m *GRPCClient) MockConnect(req *ConnectReq, callback ProviderCallback) (*ConnectRes, error) {
	m.connect(req, callback)
	return m.client.MockConnect(context.Background(), req)
}

func (m *GRPCClient) Shutdown(req *ShutdownReq) (*ShutdownRes, error) {
	return m.client.Shutdown(context.Background(), req)
}

func (m *GRPCClient) GetData(req *DataReq) (*DataRes, error) {
	return m.client.GetData(context.Background(), req)
}

func (m *GRPCClient) StoreData(req *StoreReq) (*StoreRes, error) {
	return m.client.StoreData(context.Background(), req)
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl   ProviderPlugin
	broker *plugin.GRPCBroker
	UnimplementedProviderPluginServer
}

func (m *GRPCServer) Heartbeat(ctx context.Context, req *HeartbeatReq) (*HeartbeatRes, error) {
	return m.Impl.Heartbeat(req)
}

func (m *GRPCServer) ParseCLI(ctx context.Context, req *ParseCLIReq) (*ParseCLIRes, error) {
	return m.Impl.ParseCLI(req)
}

func (m *GRPCServer) Connect(ctx context.Context, req *ConnectReq) (*ConnectRes, error) {
	conn, err := m.broker.Dial(req.CallbackServer)
	if err != nil {
		return nil, err
	}

	// Note: we do not close the connection from this side. It will get closed
	// when the plugin caller decides to kill the process.

	a := &GRPCProviderCallbackClient{NewProviderCallbackClient(conn)}
	return m.Impl.Connect(req, a)
}

func (m *GRPCServer) Disconnect(ctx context.Context, req *DisconnectReq) (*DisconnectRes, error) {
	return m.Impl.Disconnect(req)
}

func (m *GRPCServer) MockConnect(ctx context.Context, req *ConnectReq) (*ConnectRes, error) {
	conn, err := m.broker.Dial(req.CallbackServer)
	if err != nil {
		return nil, err
	}

	// Note: we do not close the connection from this side. It will get closed
	// when the plugin caller decides to kill the process.

	a := &GRPCProviderCallbackClient{NewProviderCallbackClient(conn)}
	return m.Impl.MockConnect(req, a)
}

func (m *GRPCServer) Shutdown(ctx context.Context, req *ShutdownReq) (*ShutdownRes, error) {
	return m.Impl.Shutdown(req)
}

func (m *GRPCServer) GetData(ctx context.Context, req *DataReq) (*DataRes, error) {
	resp, err := m.Impl.GetData(req)
	if err != nil {
		return nil, err
	}
	sanitizeDataRes(resp)
	return resp, nil
}

func (m *GRPCServer) StoreData(ctx context.Context, req *StoreReq) (*StoreRes, error) {
	return m.Impl.StoreData(req)
}

// GRPCClient is an implementation of ProviderCallback that talks over RPC.
type GRPCProviderCallbackClient struct{ client ProviderCallbackClient }

func (m *GRPCProviderCallbackClient) Collect(req *DataRes) error {
	_, err := m.client.Collect(context.Background(), req)
	return err
}

func (m *GRPCProviderCallbackClient) GetRecording(req *DataReq) (*ResourceData, error) {
	return m.client.GetRecording(context.Background(), req)
}

func (m *GRPCProviderCallbackClient) GetData(req *DataReq) (*DataRes, error) {
	resp, err := m.client.GetData(context.Background(), req)
	if err != nil {
		return nil, err
	}
	sanitizeDataRes(resp)
	return resp, nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCProviderCallbackServer struct {
	// This is the real implementation
	Impl ProviderCallback
	UnsafeProviderCallbackServer
}

var empty CollectRes

func (m *GRPCProviderCallbackServer) Collect(ctx context.Context, req *DataRes) (resp *CollectRes, err error) {
	return &empty, m.Impl.Collect(req)
}

func (m *GRPCProviderCallbackServer) GetRecording(ctx context.Context, req *DataReq) (resp *ResourceData, err error) {
	return m.Impl.GetRecording(req)
}

func (m *GRPCProviderCallbackServer) GetData(ctx context.Context, req *DataReq) (resp *DataRes, err error) {
	resp, err = m.Impl.GetData(req)
	if err != nil {
		return nil, err
	}
	sanitizeDataRes(resp)
	return resp, nil
}

func sanitizeDataRes(res *DataRes) {
	if res == nil {
		return
	}
	res.Error = sanitizeUtf8(res.Error)
	if res.Data == nil {
		return
	}
}

func sanitizeUtf8(s string) string {
	// This is a workaround for the fact that some providers
	// return can non-UTF8 strings on Windows. We need to sanitize them
	// to avoid issues with gRPC serialization.
	if utf8.ValidString(s) {
		return s
	}
	var buf bytes.Buffer
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid byte, skip it
			i++
			continue
		}
		buf.WriteRune(r)
		i += size
	}
	return buf.String()
}
