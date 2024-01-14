// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/terraform/connection"
	"go.mondoo.com/cnquery/v10/providers/terraform/resources"
)

const (
	// asset connection types, used internally
	StateConnectionType  = "terraform-state"
	PlanConnectionType   = "terraform-plan"
	HclConnectionType    = "terraform-hcl"
	HclGitConnectionType = "terraform-hcl-git"
	// CLI keywords, e.g. `<binary> run terraform plan file.json ...`
	CLIPlan  = "plan"
	CLIHcl   = "hcl"
	CLIState = "state"
)

type Service struct {
	plugin.Service
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	// TODO: somewhere here, parse the args for the previous sub-commands
	// perhaps set the conn.Type here depending on the args (like in the os provider)
	// and later on decide which thing to call based on the conn.Type
	// below in this file we already have something similar:
	// tc.Options["asset-type"] == "state"
	action := req.Args[0]
	switch action {
	case CLIPlan, CLIHcl, CLIState:
		switch action {
		case CLIPlan:
			conf.Type = PlanConnectionType
		case CLIHcl:
			conf.Type = HclConnectionType
		case CLIState:
			conf.Type = StateConnectionType
		}

		if len(req.Args) < 2 {
			return nil, errors.New("no path provided")
		}
		conf.Options["path"] = req.Args[1]

	default:
		if len(req.Args) > 1 {
			return nil, errors.New("unknown set of arguments, use 'state <path>', 'plan <path>' or 'hcl <path>'")
		}
		conf.Type = HclConnectionType
		conf.Options["path"] = req.Args[0]
	}

	if x, ok := flags["ignore-dot-terraform"]; ok {
		if x != nil {
			conf.Options["ignore-dot-terraform"] = strconv.FormatBool(x.RawData().Value.(bool))
			log.Info().Msg("user requested to ignore .terraform")
		}
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	res := plugin.ParseCLIRes{
		Asset: asset,
	}

	return &res, nil
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	for _, runtime := range s.runtimes {
		if conn, ok := runtime.Connection.(*connection.Connection); ok {
			conn.Close()
		}
	}
	return &plugin.ShutdownRes{}, nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
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

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn *connection.Connection
	var err error

	switch conf.Type {
	case HclConnectionType:
		s.lastConnectionID++
		conn, err = connection.NewHclConnection(s.lastConnectionID, asset)
		if err != nil {
			return nil, err
		}

	case StateConnectionType:
		s.lastConnectionID++
		conn, err = connection.NewStateConnection(s.lastConnectionID, asset)
		if err != nil {
			return nil, err
		}

	case PlanConnectionType:
		s.lastConnectionID++
		conn, err = connection.NewPlanConnection(s.lastConnectionID, asset)
		if err != nil {
			return nil, err
		}

	case HclGitConnectionType:
		s.lastConnectionID++
		conn, err = connection.NewHclGitConnection(s.lastConnectionID, asset)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("cannot find connection type " + conf.Type)
	}

	var upstream *upstream.UpstreamClient
	if req.Upstream != nil && !req.Upstream.Incognito {
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
	return &plugin.StoreRes{}, nil
}
