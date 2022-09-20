// Interface to use cnquery as a plugin
package shared

import (
	"github.com/hashicorp/go-plugin"
	"go.mondoo.com/cnquery/shared/proto"
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
	"cnquery": &CNQueryPlugin{},
}

type OutputHelper interface {
	WriteString(string) error
	Write([]byte) (int, error)
}

// CNQuery is the interface that we're exposing as a plugin.
type CNQuery interface {
	RunQuery(conf *proto.RunQueryConfig, out OutputHelper) error
}

// This is the implementation of plugin.Plugin so we can serve/consume this.
// We also implement GRPCPlugin so that this plugin can be served over
// gRPC.
type CNQueryPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl CNQuery
}

func (p *CNQueryPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterCNQueryServer(s, &GRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})
	return nil
}

func (p *CNQueryPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: proto.NewCNQueryClient(c),
		broker: broker,
	}, nil
}

var _ plugin.GRPCPlugin = &CNQueryPlugin{}
