package provider

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

type Service struct{}

func (s *Service) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	return nil, errors.New("Not yet implemented ParseCLI in os ...")
}

func (s *Service) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	return nil, errors.New("Not yet implemented Connect in os ...")
}

func (s *Service) GetData(req *proto.DataReq, callback plugin.ProviderCallback) (*llx.Result, error) {
	return nil, errors.New("Not yet implemented GetData in os ...")
}
