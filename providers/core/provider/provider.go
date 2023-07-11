package provider

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/core/resources"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

const defaultConnection uint32 = 1

type Service struct {
	runtime *plugin.Runtime
}

func Init() *Service {
	return &Service{
		runtime: &plugin.Runtime{
			Connection: defaultConnection,
			Resources:  map[string]plugin.Resource{},
		},
	}
}

func (s *Service) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	return nil, errors.New("core doesn't offer any connectors")
}

func (s *Service) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	return &proto.Connection{
		Id:   defaultConnection,
		Name: "core",
	}, nil
}

func (s *Service) GetData(req *proto.DataReq, callback plugin.ProviderCallback) (*proto.DataRes, error) {
	args, err := plugin.ProtoArgsToRawArgs(req.Args)
	if err != nil {
		return nil, err
	}

	if req.ResourceId == "" && req.Field == "" {
		res, err := resources.CreateResource(s.runtime, req.Resource, args)
		if err != nil {
			return nil, err
		}

		name := res.MqlName()
		id := res.MqlID()
		s.runtime.Resources[name+"\x00"+id] = res
		rd := llx.ResourceData(res, name).Result()
		return &proto.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := s.runtime.Resources[req.Resource+"\x00"+req.ResourceId]
	if !ok {
		return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
	}

	return resources.GetData(resource, req.Field, args), nil
}

func (s *Service) StoreData(req *proto.StoreReq) (*proto.StoreRes, error) {
	var errs []string
	for i := range req.Resources {
		info := req.Resources[i]

		args, err := plugin.ProtoArgsToRawArgs(info.Fields)
		if err != nil {
			errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), failed to parse arguments")
			continue
		}

		resource, ok := s.runtime.Resources[info.Name+"\x00"+info.Id]
		if !ok {
			resource, err = resources.CreateResource(s.runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			s.runtime.Resources[info.Name+"\x00"+info.Id] = resource
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
