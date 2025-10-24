package provider

import (
	"context"
	"errors"
	"os"
	"strings"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v12/providers/virustotal/config"
	"go.mondoo.com/cnquery/v12/providers/virustotal/connection"
	"go.mondoo.com/cnquery/v12/providers/virustotal/resources"
)

const (
	DefaultConnectionType = "virustotal"
)

type Service struct {
	*plugin.Service
}

func Init() *Service {
	return &Service{
		Service: plugin.NewService(),
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

	apiKey := ""
	if flag, ok := flags["api-key"]; ok && len(flag.Value) != 0 {
		apiKey = string(flag.Value)
	}

	if apiKey == "" {
		apiKey = firstNonEmpty(
			os.Getenv("VIRUSTOTAL_API_KEY"),
			os.Getenv("VT_API_KEY"),
		)
	}

	if apiKey != "" {
		conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential("", apiKey))
	}

	asset := &inventory.Asset{
		Name:        "VirusTotal",
		Connections: []*inventory.Config{conf},
		Labels: map[string]string{
			"cnquery.mondoo.com/provider": config.Config.ID,
		},
	}

	return &plugin.ParseCLIRes{Asset: asset}, nil
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	conn, err := s.connect(req, callback)
	if err != nil {
		return nil, err
	}

	if req.Asset.Platform == nil {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:    conn.ID(),
		Name:  conn.Name(),
		Asset: req.Asset,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.VirustotalConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewVirustotalConnection(connId, asset, conf)
		if err != nil {
			return nil, err
		}

		var upstreamClient *upstream.UpstreamClient
		if req.Upstream != nil && !req.Upstream.Incognito {
			upstreamClient, err = req.Upstream.InitClient(context.Background())
			if err != nil {
				return nil, err
			}
		}

		asset.Connections[0].Id = conn.ID()

		return plugin.NewRuntime(
			conn,
			callback,
			req.HasRecording,
			resources.CreateResource,
			resources.NewResource,
			resources.GetData,
			resources.SetData,
			upstreamClient,
		), nil
	})
	if err != nil {
		return nil, err
	}

	return runtime.Connection.(*connection.VirustotalConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.VirustotalConnection) error {
	asset.Id = conn.Identifier()
	asset.Name = "VirusTotal"

	platform, err := conn.PlatformInfo()
	if err != nil {
		return err
	}

	asset.Platform = platform
	asset.PlatformIds = []string{conn.Identifier()}
	return nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not implemented for virustotal provider")
}

func firstNonEmpty(values ...string) string {
	for _, val := range values {
		if strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}
