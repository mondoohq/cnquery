package main

import (
	"os"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/plugin"
	"go.mondoo.com/cnquery/providers/proto"
)

func main() {
	plugin.Start(os.Args, &server{})
}

type server struct{}

func (s *server) ParseCLI(req *proto.ParseCLIReq) (*proto.ParseCLIRes, error) {
	panic("NOT YET FOR OS")
}

func (s *server) Connect(req *proto.ConnectReq) (*proto.Connection, error) {
	panic("NOT YET FOR OS")
}

func (s *server) GetData(req *proto.DataReq, callback plugin.ProviderCallback) (*llx.Result, error) {
	panic("NOT YET FOR OS")
}
