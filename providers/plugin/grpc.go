package plugin

import (
	plugin "github.com/hashicorp/go-plugin"
	"go.mondoo.com/cnquery/providers/proto"
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
	client proto.ProviderPluginClient
}

func (m *GRPCClient) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	return m.client.ParseCLI(context.Background(), req)
}

func (m *GRPCClient) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	return m.client.Connect(context.Background(), req)
}

func (m *GRPCClient) GetData(req *proto.DataReq, callback ProviderCallback) (*proto.DataRes, error) {
	helper := &GRPCProviderCallbackServer{Impl: callback}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		proto.RegisterProviderCallbackServer(s, helper)

		return s
	}

	brokerID := m.broker.NextId()
	req.CallbackServer = brokerID
	go m.broker.AcceptAndServe(brokerID, serverFunc)

	res, err := m.client.GetData(context.Background(), req)

	s.Stop()
	return res, err
}

func (m *GRPCClient) StoreData(req *proto.StoreReq) (*proto.StoreRes, error) {
	return m.client.StoreData(context.Background(), req)
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl   ProviderPlugin
	broker *plugin.GRPCBroker
	proto.UnimplementedProviderPluginServer
}

func (m *GRPCServer) ParseCLI(ctx context.Context, req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	return m.Impl.ParseCLI(req)
}

func (m *GRPCServer) Connect(ctx context.Context, req *proto.ConnectReq) (*proto.Connection, error) {
	return m.Impl.Connect(req)
}

func (m *GRPCServer) GetData(ctx context.Context, req *proto.DataReq) (*proto.DataRes, error) {
	conn, err := m.broker.Dial(req.CallbackServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	a := &GRPCProviderCallbackClient{proto.NewProviderCallbackClient(conn)}
	return m.Impl.GetData(req, a)
}

func (m *GRPCServer) StoreData(ctx context.Context, req *proto.StoreReq) (*proto.StoreRes, error) {
	return m.Impl.StoreData(req)
}

// GRPCClient is an implementation of ProviderCallback that talks over RPC.
type GRPCProviderCallbackClient struct{ client proto.ProviderCallbackClient }

func (m *GRPCProviderCallbackClient) Collect(req *proto.DataRes) error {
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

func (m *GRPCProviderCallbackServer) Collect(ctx context.Context, req *proto.DataRes) (resp *proto.CollectRes, err error) {
	return &empty, m.Impl.Collect(req)
}
