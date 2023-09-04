// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources"
	"go.mondoo.com/cnquery/providers/os/resources/discovery/container_registry"
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

	conf := &inventory.Config{
		Sudo:     shared.ParseSudo(flags),
		Discover: parseDiscover(flags),
		Type:     req.Connector,
	}

	port := 0
	containerID := ""
	switch req.Connector {
	case "local":
		conf.Type = "local"
	case "ssh":
		conf.Type = "ssh"
		port = 22
	case "winrm":
		conf.Type = "winrm"
		port = 5985
	case "vagrant":
		conf.Type = "vagrant"
	case "container", "docker":
		if len(req.Args) > 1 {
			switch req.Args[0] {
			case "image":
				conf.Type = "docker-image"
				conf.Backend = "tar"
				conf.Host = req.Args[1]
			case "registry":
				conf.Type = "docker-registry"
				conf.Host = req.Args[1]
			case "tar":
				conf.Type = "docker-snapshot"
				conf.Path = req.Args[1]
			}
		} else {
			conf.Type = "docker-container"
			containerID = req.Args[0]
		}
	case "filesystem", "fs":
		conf.Type = "filesystem"
	}

	user := ""
	if len(req.Args) != 0 && !(strings.HasPrefix(req.Connector, "docker") || strings.HasPrefix(req.Connector, "container")) {
		target := req.Args[0]
		if !strings.Contains(target, "://") {
			target = "ssh://" + target
		}

		x, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("incorrect format of target, please use user@host:port")
		}

		user = x.User.Username()
		conf.Host = x.Hostname()
		conf.Path = x.Path

		if sPort := x.Port(); sPort != "" {
			port, err = strconv.Atoi(x.Port())
			if err != nil {
				return nil, errors.New("port '" + x.Port() + "'is incorrectly formatted, must be a number")
			}
		}
	}

	if port > 0 {
		conf.Port = int32(port)
	}

	if x, ok := flags["password"]; ok && len(x.Value) != 0 {
		conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential(user, string(x.Value)))
	}

	if x, ok := flags["identity-file"]; ok && len(x.Value) != 0 {
		credential, err := vault.NewPrivateKeyCredentialFromPath(user, string(x.Value), "")
		if err != nil {
			return nil, err
		}
		conf.Credentials = append(conf.Credentials, credential)
	}

	if x, ok := flags["path"]; ok && len(x.Value) != 0 {
		conf.Path = string(x.Value)
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	if containerID != "" {
		asset.Name = containerID
		conf.Host = containerID
	}

	idDetector := "hostname"
	if flag, ok := flags["id-detector"]; ok {
		if string(flag.Value) != "" {
			idDetector = string(flag.Value)
		}
	}
	if idDetector != "" {
		asset.IdDetector = []string{idDetector}
	}

	res := plugin.ParseCLIRes{
		Asset: asset,
	}

	return &res, nil
}

// LocalAssetReq ist a sample request to connect to the local OS.
// Useful for test automation.
var LocalAssetReq = &plugin.ConnectReq{
	Asset: &inventory.Asset{
		Connections: []*inventory.Config{{
			Type: "local",
		}},
	},
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	conn, err := s.connect(req, callback)
	if err != nil {
		return nil, err
	}

	// We only need to run the detection step when we don't have any asset information yet.
	if req.Asset.Platform == nil {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}

	var inv *inventory.Inventory
	if conn.Asset().Connections[0].Type == "docker-registry" {
		inv, err = s.discover(conn.(*connection.TarConnection))
		if err != nil {
			return nil, err
		}
	}

	if inv == nil {
		inv = &inventory.Inventory{
			Spec: &inventory.InventorySpec{
				Assets: []*inventory.Asset{req.Asset},
			},
		}
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inv,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn shared.Connection
	var err error

	switch conf.Type {
	case "local":
		s.lastConnectionID++
		conn = connection.NewLocalConnection(s.lastConnectionID, conf, asset)

	case "ssh":
		s.lastConnectionID++
		conn, err = connection.NewSshConnection(s.lastConnectionID, conf, asset)

	case "mock":
		s.lastConnectionID++
		conn, err = mock.New("", asset)

	case "tar":
		s.lastConnectionID++
		conn, err = connection.NewTarConnection(s.lastConnectionID, conf, asset)

	case "docker-snapshot":
		s.lastConnectionID++
		conn, err = connection.NewDockerSnapshotConnection(s.lastConnectionID, conf, asset)

	case "vagrant":
		s.lastConnectionID++
		conn, err = connection.NewVagrantConnection(s.lastConnectionID, conf, asset)
		if err != nil {
			return nil, err
		}
		// We need to detect the platform for the connection asset here, because
		// this platform information will be used to determine the package manager
		err := s.detect(conn.Asset(), conn)
		if err != nil {
			return nil, err
		}

	case "docker-image":
		s.lastConnectionID++
		conn, err = connection.NewDockerContainerImageConnection(s.lastConnectionID, conf, asset)

	case "docker-container":
		s.lastConnectionID++
		conn, err = connection.NewDockerEngineContainer(s.lastConnectionID, conf, asset)

	case "docker-registry", "container-registry":
		s.lastConnectionID++
		conn, err = connection.NewContainerRegistryImage(s.lastConnectionID, conf, asset)

	case "registry-image":
		s.lastConnectionID++
		conn, err = connection.NewContainerRegistryImage(s.lastConnectionID, conf, asset)

	case "filesystem":
		s.lastConnectionID++
		conn, err = connection.NewFileSystemConnection(s.lastConnectionID, conf, asset)

	default:
		return nil, errors.New("cannot find connection type " + conf.Type)
	}

	if err != nil {
		return nil, err
	}

	var upstream *upstream.UpstreamClient
	if req.Upstream != nil {
		upstream, err = req.Upstream.InitClient()
		if err != nil {
			return nil, err
		}
	}

	asset.Connections[0].Id = conn.ID()
	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection:     conn,
		Callback:       callback,
		HasRecording:   req.HasRecording,
		CreateResource: resources.CreateResource,
		Upstream:       upstream,
	}

	return conn, err
}

func (s *Service) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
	}

	args := plugin.PrimitiveArgsToRawDataArgs(req.Args, runtime)

	if req.ResourceId == "" && req.Field == "" {
		res, err := resources.NewResource(runtime, req.Resource, args)
		if err != nil {
			return nil, err
		}

		rd := llx.ResourceData(res, res.MqlName()).Result()
		return &plugin.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources.Get(req.Resource + "\x00" + req.ResourceId)
	if !ok {
		// Note: Since resources are internally always created, there are only very
		// few cases where we arrive here:
		// 1. The caller is wrong. Possibly a mixup with IDs
		// 2. The resource was loaded from a recording, but the field is not
		// in the recording. Thus the resource was never created inside the
		// plugin. We will attempt to create the resource and see if the field
		// can be computed.
		if !runtime.HasRecording {
			return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
		}

		args, err := runtime.ResourceFromRecording(req.Resource, req.ResourceId)
		if err != nil {
			return nil, errors.New("attempted to load resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}

		resource, err = resources.CreateResource(runtime, req.Resource, args)
		if err != nil {
			return nil, errors.New("attempted to create resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}
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

		args, err := plugin.ProtoArgsToRawDataArgs(info.Fields)
		if err != nil {
			errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), failed to parse arguments")
			continue
		}

		resource, ok := runtime.Resources.Get(info.Name + "\x00" + info.Id)
		if !ok {
			resource, err = resources.CreateResource(runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			runtime.Resources.Set(info.Name+"\x00"+info.Id, resource)
		} else {
			if err := resources.SetAllData(resource, args); err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), field error: "+err.Error())
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}
	return &plugin.StoreRes{}, nil
}

func (s *Service) discover(conn *connection.TarConnection) (*inventory.Inventory, error) {
	conf := conn.Asset().Connections[0]
	if conf == nil {
		return nil, nil
	}

	resolver := container_registry.Resolver{}
	resolvedAssets, err := resolver.Resolve(context.Background(), conn.Asset(), conf, nil)
	if err != nil {
		return nil, err
	}

	inventory := &inventory.Inventory{}
	inventory.AddAssets(resolvedAssets...)

	return inventory, nil
}
