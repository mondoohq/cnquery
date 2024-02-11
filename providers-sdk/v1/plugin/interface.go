// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Interface to use cnquery as a plugin
package plugin

import (
	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "ifenohXaoHoh3Iequeg0iuph2gaeth",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"provider": &ProviderPluginImpl{},
}

type Closer interface {
	Close()
}

type ProviderCallback interface {
	Collect(req *DataRes) error
	GetRecording(req *DataReq) (*ResourceData, error)
	GetData(req *DataReq) (*DataRes, error)
}

// ProviderPlugin is the interface that we're exposing as a plugin.
type ProviderPlugin interface {
	Heartbeat(req *HeartbeatReq) (*HeartbeatRes, error)
	ParseCLI(req *ParseCLIReq) (*ParseCLIRes, error)
	Connect(req *ConnectReq, callback ProviderCallback) (*ConnectRes, error)
	Disconnect(req *DisconnectReq) (*DisconnectRes, error)
	MockConnect(req *ConnectReq, callback ProviderCallback) (*ConnectRes, error)
	Shutdown(req *ShutdownReq) (*ShutdownRes, error)
	GetData(req *DataReq) (*DataRes, error)
	StoreData(req *StoreReq) (*StoreRes, error)
}

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type ProviderPluginImpl struct {
	plugin.NetRPCUnsupportedPlugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl ProviderPlugin
}

func (p *ProviderPluginImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterProviderPluginServer(s, &GRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *ProviderPluginImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: NewProviderPluginClient(c),
		broker: broker,
	}, nil
}

var _ plugin.GRPCPlugin = &ProviderPluginImpl{}
