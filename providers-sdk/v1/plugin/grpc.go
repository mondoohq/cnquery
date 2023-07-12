package plugin

import (
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

func (m *GRPCClient) ParseCLI(req *ParseCLIReq) (*ParseCLIRes, error) {
	return m.client.ParseCLI(context.Background(), req)
}

func (m *GRPCClient) Connect(req *ConnectReq) (*ConnectRes, error) {
	return m.client.Connect(context.Background(), req)
}

func (m *GRPCClient) GetData(req *DataReq, callback ProviderCallback) (*DataRes, error) {
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

	res, err := m.client.GetData(context.Background(), req)

	s.Stop()
	return res, err
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

func (m *GRPCServer) ParseCLI(ctx context.Context, req *ParseCLIReq) (*ParseCLIRes, error) {
	return m.Impl.ParseCLI(req)
}

func (m *GRPCServer) Connect(ctx context.Context, req *ConnectReq) (*ConnectRes, error) {
	return m.Impl.Connect(req)
}

func (m *GRPCServer) GetData(ctx context.Context, req *DataReq) (*DataRes, error) {
	conn, err := m.broker.Dial(req.CallbackServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	a := &GRPCProviderCallbackClient{NewProviderCallbackClient(conn)}
	return m.Impl.GetData(req, a)
}

func (m *GRPCServer) StoreData(ctx context.Context, req *StoreReq) (*StoreRes, error) {
	return m.Impl.StoreData(req)
}

// GRPCClient is an implementation of ProviderCallback that talks over RPC.
type GRPCProviderCallbackClient struct{ client ProviderCallbackClient }

func (m *GRPCProviderCallbackClient) Collect(req *DataRes) error {
	// _, err := m.client.Write(context.Background(), &String{
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
	UnsafeProviderCallbackServer
}

var empty CollectRes

func (m *GRPCProviderCallbackServer) Collect(ctx context.Context, req *DataRes) (resp *CollectRes, err error) {
	return &empty, m.Impl.Collect(req)
}
