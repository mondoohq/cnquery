package plugin

import (
	plugin "github.com/hashicorp/go-plugin"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/proto"
	"golang.org/x/net/context"
)

func init() {
	var x ProviderPlugin = &GRPCClient{}
	_ = x
}

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCClient struct {
	broker *plugin.GRPCBroker
	client proto.ProviderPluginClient
}

func (m *GRPCClient) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	panic("NOT IMPL")
}

func (m *GRPCClient) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	panic("NOT IMPL")
}

func (m *GRPCClient) GetData(req *proto.DataReq, callback ProviderCallback) (*llx.Result, error) {
	panic("NOT IMPL")
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl   ProviderPlugin
	broker *plugin.GRPCBroker
	proto.UnimplementedProviderPluginServer
}

func (m *GRPCServer) GetData(ctx context.Context, req *proto.DataReq) (*llx.Result, error) {
	conn, err := m.broker.Dial(req.CallbackServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	a := &GRPCProviderCallbackClient{proto.NewProviderCallbackClient(conn)}
	return m.Impl.GetData(req, a)
}

// GRPCClient is an implementation of ProviderCallback that talks over RPC.
type GRPCProviderCallbackClient struct{ client proto.ProviderCallbackClient }

func (m *GRPCProviderCallbackClient) Collect(req *llx.Result) error {
	// _, err := m.client.Write(context.Background(), &proto.String{
	// 	Data: string(b),
	// })
	// if err != nil {
	// 	hclog.Default().Info("out.Write", "client", "start", "err", err)
	// 	return 0, err
	// }
	// return 0, nil
	panic("COLLECT async")
	return nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCProviderCallbackServer struct {
	// This is the real implementation
	Impl ProviderCallback
	proto.UnsafeProviderCallbackServer
}

var empty proto.CollectRes

func (m *GRPCProviderCallbackServer) Collect(ctx context.Context, req *llx.Result) (resp *proto.CollectRes, err error) {
	return &empty, m.Impl.Collect(req)
}
