package provider

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

type Service struct {
	localConnections map[string]*connection.LocalConnection
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

	conn := asset.Connections[0]
	switch conn.Backend {
	case providers.ProviderType_LOCAL_OS:
		res := connection.NewLocalConnection(s.lastConnectionID)
		s.lastConnectionID++
		return res, nil

	default:
		return nil, errors.New("cannot find conneciton type " + conn.Backend.Id())
	}
}

func (s *Service) GetData(req *proto.DataReq, callback plugin.ProviderCallback) (*llx.Result, error) {
	return nil, errors.New("Not yet implemented GetData in os ...")
}
