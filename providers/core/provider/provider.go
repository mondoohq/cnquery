package provider

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/core/resources"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

const defaultConnection uint32 = 1

type Service struct {
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes: map[uint32]*plugin.Runtime{},
	}
}

func (s *Service) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	return nil, errors.New("core doesn't offer any connectors")
}

func (s *Service) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	if req == nil || req.Asset == nil || req.Asset.Spec == nil {
		return nil, errors.New("no connection data provided")
	}

	assets := req.Asset.Spec.Assets
	if len(assets) == 0 {
		return nil, errors.New("no asset provided for connection")
	}

	s.lastConnectionID++
	connID := s.lastConnectionID
	runtime := &plugin.Runtime{
		Resources: map[string]plugin.Resource{},
	}
	s.runtimes[connID] = runtime

	asset := req.Asset.Spec.Assets[0]
	assetObj, err := resources.CreateResource(runtime, "asset", map[string]interface{}{
		"ids":      llx.TArr2Raw(asset.PlatformIds),
		"platform": asset.Platform.Name,
		"kind":     asset.Platform.Kind.String(),
		"runtime":  asset.Platform.Runtime,
		"version":  asset.Platform.Version,
		"arch":     asset.Platform.Arch,
		"title":    asset.Platform.Title,
		"family":   llx.TArr2Raw(asset.Platform.Family),
		"build":    asset.Platform.Build,
		"labels":   llx.TMap2Raw(asset.Platform.Labels),
		"fqdn":     "",
	})
	if err != nil {
		return nil, errors.New("failed to init core, cannot set asset metadata")
	}
	runtime.Resources["asset\x00"] = assetObj

	return &proto.Connection{
		Id:   defaultConnection,
		Name: "core",
	}, nil
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
		if x, ok := runtime.Resources[name+"\x00"+id]; ok {
			res = x
		} else {
			runtime.Resources[name+"\x00"+id] = res
		}

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
