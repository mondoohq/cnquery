// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/core/resources"
	"go.mondoo.com/cnquery/v10/types"
)

const defaultConnection uint32 = 1

type Service struct {
	plugin.Service
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes: map[uint32]*plugin.Runtime{},
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	return nil, errors.New("core doesn't offer any connectors")
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	var upstream *upstream.UpstreamClient
	var err error
	if req.Upstream != nil && !req.Upstream.Incognito {
		upstream, err = req.Upstream.InitClient()
		if err != nil {
			return nil, err
		}
	}

	s.lastConnectionID++
	connID := s.lastConnectionID
	runtime := &plugin.Runtime{
		Callback:     callback,
		Upstream:     upstream,
		HasRecording: req.HasRecording,
	}
	s.runtimes[connID] = runtime

	asset := req.Asset
	_, err = resources.CreateResource(runtime, "asset", map[string]*llx.RawData{
		"ids":      llx.ArrayData(llx.TArr2Raw(asset.PlatformIds), types.String),
		"platform": llx.StringData(asset.Platform.Name),
		"name":     llx.StringData(asset.Name),
		"kind":     llx.StringData(asset.Platform.Kind),
		"runtime":  llx.StringData(asset.Platform.Runtime),
		"version":  llx.StringData(asset.Platform.Version),
		"arch":     llx.StringData(asset.Platform.Arch),
		"title":    llx.StringData(asset.Platform.PrettyTitle()),
		"family":   llx.ArrayData(llx.TArr2Raw(asset.Platform.Family), types.String),
		"build":    llx.StringData(asset.Platform.Build),
		"labels":   llx.MapData(llx.TMap2Raw(asset.Platform.Labels), types.String),
		"fqdn":     llx.StringData(asset.Fqdn),
	})
	if err != nil {
		return nil, errors.New("failed to init core, cannot set asset metadata")
	}

	if len(asset.Connections) > 0 {
		_, err = resources.CreateResource(runtime, "mondoo", map[string]*llx.RawData{
			"capabilities": llx.ArrayData(llx.TArr2Raw(asset.Connections[0].Capabilities), types.String),
		})
		if err != nil {
			return nil, errors.New("failed to init core, cannot set connection metadata")
		}
	}

	return &plugin.ConnectRes{
		Id:   connID,
		Name: "core",
	}, nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return s.Connect(req, callback)
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	return &plugin.ShutdownRes{}, nil
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

		rd := llx.ResourceData(res, req.Resource).Result()
		return &plugin.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources.Get(req.Resource + "\x00" + req.ResourceId)
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
