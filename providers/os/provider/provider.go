package provider

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/asset"
	v1 "go.mondoo.com/cnquery/motor/inventory/v1"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

type Service struct {
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}
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
		Sudo:     shared.ParseSudo(flags),
		Discover: parseDiscover(flags),
	}

	port := 0
	switch req.Connector {
	case "local":
		conn.Backend = providers.ProviderType_LOCAL_OS
	case "ssh":
		conn.Backend = providers.ProviderType_SSH
		port = 22
	case "winrm":
		conn.Backend = providers.ProviderType_WINRM
		port = 5985
	}

	user := ""
	if len(req.Args) != 0 {
		target := req.Args[0]
		if !strings.Contains(target, "://") {
			target = "ssh://" + target
		}

		x, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("incorrect format of target, please use user@host:port")
		}

		user = x.User.Username()
		conn.Host = x.Hostname()
		conn.Path = x.Path

		if sPort := x.Port(); sPort != "" {
			port, err = strconv.Atoi(x.Port())
			if err != nil {
				return nil, errors.New("port '" + x.Port() + "'is incorrectly formatted, must be a number")
			}
		}
	}

	if port > 0 {
		conn.Port = int32(port)
	}

	if x, ok := flags["password"]; ok && len(x.Value) != 0 {
		conn.Credentials = append(conn.Credentials, vault.NewPasswordCredential(user, string(x.Value)))
	}

	assets, err := s.Resolve(&asset.Asset{
		Connections: []*providers.Config{conn},
	})
	if err != nil {
		return nil, errors.New("failed to resolve: " + err.Error())
	}

	idDetector := string(flags["id-detector"].Value)
	if idDetector != "" {
		for i := range assets {
			assets[i].IdDetector = []string{idDetector}
		}
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

func (s *Service) connect(asset *asset.Asset) (shared.Connection, error) {
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	var conn shared.Connection
	var err error
	conf := asset.Connections[0]
	switch conf.Backend {
	case providers.ProviderType_LOCAL_OS:
		conn = connection.NewLocalConnection(s.lastConnectionID)
		s.lastConnectionID++

	case providers.ProviderType_SSH:
		conn, err = connection.NewSshConnection(s.lastConnectionID, conf)
		s.lastConnectionID++

	default:
		return nil, errors.New("cannot find conneciton type " + conf.Backend.Id())
	}

	if err != nil {
		return nil, err
	}

	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection: conn,
		Resources:  map[string]plugin.Resource{},
	}

	return conn, err
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
		id := res.MqlID()
		runtime.Resources[name+"\x00"+id] = res
		rd := llx.ResourceData(res, name).Result()
		return &proto.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources[req.Resource+"\x00"+req.ResourceId]
	if !ok {
		return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
	}

	return resources.GetData(resource, req.Field, args), nil
}

func (s *Service) StoreData(req *proto.StoreReq) (*proto.StoreRes, error) {
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
	}

	var errs []string
	for i := range req.Resources {
		info := req.Resources[i]

		args, err := plugin.ProtoArgsToRawArgs(info.Fields)
		if err != nil {
			errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), failed to parse arguments")
			continue
		}

		resource, ok := runtime.Resources[info.Name+"\x00"+info.Id]
		if !ok {
			resource, err = resources.CreateResource(runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			runtime.Resources[info.Name+"\x00"+info.Id] = resource
		}

		for k, v := range args {
			if err := resources.SetData(resource, k, v); err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), field error: "+err.Error())
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}
	return nil, nil
}
