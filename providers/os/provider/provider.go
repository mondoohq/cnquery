package provider

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources"
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

func parseDiscover(flags map[string]*llx.Primitive) *inventory.Discovery {
	// TODO: parse me...
	return &inventory.Discovery{Targets: []string{"auto"}}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conn := &inventory.Config{
		Sudo:     shared.ParseSudo(flags),
		Discover: parseDiscover(flags),
		Type:     req.Connector,
	}

	port := 0
	switch req.Connector {
	case "local":
		conn.Type = "local"
	case "ssh":
		conn.Type = "ssh"
		port = 22
	case "winrm":
		conn.Type = "winrm"
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

	assets, err := s.resolve(&inventory.Asset{
		Connections: []*inventory.Config{conn},
	})
	if err != nil {
		return nil, errors.New("failed to resolve: " + err.Error())
	}

	idDetector := "hostname"
	if flag, ok := flags["id-detector"]; ok {
		idDetector = string(flag.Value)
	}
	if idDetector != "" {
		for i := range assets {
			assets[i].IdDetector = []string{idDetector}
		}
	}

	res := plugin.ParseCLIRes{
		Inventory: &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: assets,
			},
		},
	}

	return &res, nil
}

func (s *Service) resolve(rootAsset *inventory.Asset) ([]*inventory.Asset, error) {
	res := []*inventory.Asset{rootAsset}

	if err := s.detect(rootAsset); err != nil {
		return nil, err
	}

	// TODO: discovery of related assets

	return res, nil
}

// LocalAssetReq ist a sample request to connect to the local OS.
// Useful for test automation.
var LocalAssetReq = &plugin.ConnectReq{
	Asset: &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{{
				Connections: []*inventory.Config{{
					Type: "local",
				}},
			}},
		},
	},
}

func (s *Service) Connect(req *plugin.ConnectReq) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil || req.Asset.Spec == nil {
		return nil, errors.New("no connection data provided")
	}

	assets := req.Asset.Spec.Assets
	if len(assets) != 1 {
		return nil, errors.New("too many assets provided in connection")
	}

	conn, err := s.connect(assets[0])
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:   uint32(conn.ID()),
		Name: conn.Name(),
	}, nil
}

func (s *Service) connect(asset *inventory.Asset) (shared.Connection, error) {
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	conf := asset.Connections[0]
	var conn shared.Connection
	var err error

	switch conf.Type {
	case "local":
		s.lastConnectionID++
		conn = connection.NewLocalConnection(s.lastConnectionID)

	case "ssh":
		s.lastConnectionID++
		conn, err = connection.NewSshConnection(s.lastConnectionID, conf)

	case "mock":
		s.lastConnectionID++
		conn, err = mock.New("")

	default:
		return nil, errors.New("cannot find connection type " + conf.Type)
	}

	if err != nil {
		return nil, err
	}

	asset.Connections[0].Id = conn.ID()
	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection: conn,
		Resources:  map[string]plugin.Resource{},
	}

	return conn, err
}

func (s *Service) GetData(req *plugin.DataReq, callback plugin.ProviderCallback) (*plugin.DataRes, error) {
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

		rd := llx.ResourceData(res, res.MqlName()).Result()
		return &plugin.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources[req.Resource+"\x00"+req.ResourceId]
	if !ok {
		return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
	}

	return resources.GetData(resource, req.Field, args), nil
}

func (s *Service) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
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
