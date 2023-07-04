package provider

import (
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/resources"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

type Service struct {
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func parseDiscover(flags map[string]*llx.Primitive) *providers.Discovery {
	// TODO: parse me...
	return &providers.Discovery{Targets: []string{"auto"}}
}

func (s *Service) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conn := &providers.Config{
		Sudo:     connection.ParseSudo(flags),
		Discover: parseDiscover(flags),
	}

	switch req.Connector {
	case "local":
		conn.Backend = providers.ProviderType_LOCAL_OS
	case "ssh":
		conn.Backend = providers.ProviderType_SSH
	case "winrm":
		conn.Backend = providers.ProviderType_WINRM
	}

	assets, err := s.Resolve(&asset.Asset{
		Connections: []*providers.Config{conn},
	})
	if err != nil {
		return nil, errors.New("failed to resolve: " + err.Error())
	}

	res := proto.ParseCLIRes{
		Inventory: &v1.Inventory{
			Spec: &v1.InventorySpec{
				Assets: assets,
			},
		},
	}

	return &res, nil
}

func (s *Service) Resolve(rootAsset *asset.Asset) ([]*asset.Asset, error) {
	obj := &asset.Asset{
		Name:        rootAsset.Mrn,
		State:       asset.State_STATE_ONLINE,
		Connections: rootAsset.Connections,
	}

	if err := s.detect(obj); err != nil {
		return nil, err
	}

	res := []*asset.Asset{obj}

	// TODO: discovery of other related assets

	return res, nil
}

func (s *Service) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	if req == nil || req.Asset == nil || req.Asset.Spec == nil {
		return nil, errors.New("no connection data provided")
	}

	assets := req.Asset.Spec.Assets
	if len(assets) == 0 {
		return nil, errors.New("no assets provided in connection")
	}
	if len(assets) != 1 {
		return nil, errors.New("too many assets provided in connection")
	}

	asset := assets[0]
	conn, err := s.connect(asset)
	if err != nil {
		return nil, err
	}

	return &proto.Connection{Id: uint32(conn.ID())}, nil
}

func (s *Service) connect(asset *asset.Asset) (connection.Connection, error) {
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	var conn connection.Connection
	conf := asset.Connections[0]
	switch conf.Backend {
	case providers.ProviderType_LOCAL_OS:
		conn = connection.NewLocalConnection(s.lastConnectionID)
		s.lastConnectionID++

	default:
		return nil, errors.New("cannot find conneciton type " + conf.Backend.Id())
	}

	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection: conn,
		Resources:  map[string]plugin.Resource{},
	}

	return conn, nil
}

func (s *Service) GetData(req *proto.DataReq, callback plugin.ProviderCallback) (*proto.DataRes, error) {
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
	}

	args, err := plugin.ProtoArgsToRawArgs(req.Args)
	if err != nil {
		return nil, err
	}

	if req.ResourceId == "" && req.Field == "" {
		res, err := resources.CreateResource(runtime, req.Resource, args)
		if err != nil {
			return nil, err
		}

		name := res.MqlName()
		id, err := res.MqlID()
		if err != nil {
			return nil, errors.New("failed to create resource " + name + ", ID returned an error: " + err.Error())
		}
		runtime.Resources[name+"\x00"+id] = res
		rd := llx.ResourceData(res, name).Result()
		return &proto.DataRes{
			Data: rd.Data,
		}, nil
	}

	return nil, errors.New("Not yet implemented GetData in os ...")
}
