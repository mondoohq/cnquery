// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container"
	"go.mondoo.com/cnquery/v10/providers/os/connection/docker"
	"go.mondoo.com/cnquery/v10/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v10/providers/os/connection/local"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/ssh"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
	"go.mondoo.com/cnquery/v10/providers/os/connection/vagrant"
	"go.mondoo.com/cnquery/v10/providers/os/connection/winrm"
	"go.mondoo.com/cnquery/v10/providers/os/id"
	"go.mondoo.com/cnquery/v10/providers/os/resources"
	"go.mondoo.com/cnquery/v10/providers/os/resources/discovery/container_registry"
	"go.mondoo.com/cnquery/v10/providers/os/resources/discovery/docker_engine"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

type Service struct {
	*plugin.Service
}

func Init() *Service {
	return &Service{
		Service: plugin.NewService(),
	}
}

func parseDiscover(flags map[string]*llx.Primitive) *inventory.Discovery {
	discovery := &inventory.Discovery{Targets: []string{"auto"}}
	if flag, ok := flags["discover"]; ok && len(flag.Array) > 0 {
		discovery.Targets = []string{}
		for i := range flag.Array {
			discovery.Targets = append(discovery.Targets, string(flag.Array[i].Value))
		}
	}
	return discovery
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
	case "docker":
		if len(req.Args) > 1 {
			switch req.Args[0] {
			case "image":
				conf.Type = "docker-image"
				conf.Host = req.Args[1]
			case "registry":
				conf.Type = "docker-registry"
				conf.Host = req.Args[1]
			case "tar":
				conf.Type = "docker-snapshot"
				conf.Path = req.Args[1]
			case "container":
				conf.Type = "docker-container"
				conf.Host = req.Args[1]
			}
		} else {
			connType, err := docker.FindDockerObjectConnectionType(req.Args[0])
			if err != nil {
				return nil, err
			}
			conf.Type = connType
			containerID = req.Args[0]
		}
	case "container":
		if len(req.Args) > 1 {
			switch req.Args[0] {
			case "image":
				conf.Type = "docker-image"
				conf.Host = req.Args[1]
			case "registry":
				conf.Type = "docker-registry"
				conf.Host = req.Args[1]
			case "tar":
				conf.Type = "docker-snapshot"
				conf.Path = req.Args[1]
			case "container":
				conf.Type = "docker-container"
				conf.Host = req.Args[1]
			}
		} else {
			connType := identifyContainerType(req.Args[0])
			conf.Type = connType
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

	identityFileProvided := false
	if x, ok := flags["identity-file"]; ok && len(x.Value) != 0 {
		credential, err := vault.NewPrivateKeyCredentialFromPath(user, string(x.Value), "")
		if err != nil {
			return nil, err
		}
		conf.Credentials = append(conf.Credentials, credential)
		identityFileProvided = true
	}

	if x, ok := flags["path"]; ok && len(x.Value) != 0 {
		conf.Path = string(x.Value)
	}

	if user != "" && !identityFileProvided {
		conf.Credentials = append(conf.Credentials, &vault.Credential{Type: vault.CredentialType_ssh_agent, User: user})
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	if containerID != "" {
		asset.Name = containerID
		conf.Host = containerID
	}

	idDetector := ""
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
	if !req.Asset.Connections[0].SkipDiscovery && (req.Asset.Platform == nil || req.Asset.Platform.Name == "") {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}
	log.Debug().Str("asset", req.Asset.Name).Msg("detected asset")

	var inv *inventory.Inventory
	connType := conn.Asset().Connections[0].Type
	switch connType {
	case "docker-registry":
		tarConn := conn.(*tar.Connection)
		inv, err = s.discoverRegistry(tarConn)
		if err != nil {
			return nil, err
		}
	case "local", "docker-container":
		inv, err = s.discoverLocalContainers(conn.Asset().Connections[0])
		if err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inv,
	}, nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	asset := &inventory.Asset{
		PlatformIds: req.Asset.PlatformIds,
		Platform:    req.Asset.Platform,
		Connections: []*inventory.Config{{
			Type: "mock",
		}},
	}

	conn, err := s.connect(&plugin.ConnectReq{
		Features: req.Features,
		Upstream: req.Upstream,
		Asset:    asset,
	}, callback)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:    uint32(conn.ID()),
		Name:  conn.Name(),
		Asset: asset,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		var conn shared.Connection
		var err error

		switch conf.Type {
		case shared.Type_Local.String():
			conn = local.NewConnection(connId, conf, asset)

			fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
			if err == nil {
				asset.Name = fingerprint.Name
				asset.PlatformIds = fingerprint.PlatformIDs
				asset.IdDetector = fingerprint.ActiveIdDetectors
			}

		case shared.Type_SSH.String():
			conn, err = ssh.NewConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}

			fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
			if err == nil {
				if conn.Asset().Connections[0].Runtime != "vagrant" {
					asset.Name = fingerprint.Name
				}
				asset.PlatformIds = fingerprint.PlatformIDs
				asset.IdDetector = fingerprint.ActiveIdDetectors
			}

		case shared.Type_Winrm.String():
			conn, err = winrm.NewConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}

			fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
			if err == nil {
				asset.Name = fingerprint.Name
				asset.PlatformIds = fingerprint.PlatformIDs
				asset.IdDetector = fingerprint.ActiveIdDetectors
			}

		case shared.Type_Tar.String():
			conn, err = tar.NewConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}

			fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
			if err == nil {
				asset.Name = fingerprint.Name
				asset.PlatformIds = fingerprint.PlatformIDs
				asset.IdDetector = fingerprint.ActiveIdDetectors
			}

		case shared.Type_DockerSnapshot.String():
			conn, err = docker.NewSnapshotConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}

			fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
			if err == nil {
				asset.Name = fingerprint.Name
				asset.PlatformIds = fingerprint.PlatformIDs
				asset.IdDetector = fingerprint.ActiveIdDetectors
			}

		case shared.Type_Vagrant.String():
			conn, err = vagrant.NewVagrantConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}
			// We need to detect the platform for the connection asset here, because
			// this platform information will be used to determine the package manager
			err := s.detect(conn.Asset(), conn)
			if err != nil {
				return nil, err
			}

		case shared.Type_DockerImage.String():
			conn, err = docker.NewContainerImageConnection(connId, conf, asset)

		case shared.Type_DockerContainer.String():
			conn, err = docker.NewDockerEngineContainer(connId, conf, asset)

		case shared.Type_DockerRegistry.String(), shared.Type_ContainerRegistry.String():
			conn, err = container.NewRegistryImage(connId, conf, asset)

		case shared.Type_RegistryImage.String():
			conn, err = container.NewRegistryImage(connId, conf, asset)

		case shared.Type_FileSystem.String():
			conn, err = fs.NewConnection(connId, conf, asset)
			if err != nil {
				return nil, err
			}
			// This is a workaround to set Google COS platform IDs when scanned from inside k8s
			pID, err := conn.(*fs.FileSystemConnection).Identifier()
			if err != nil {
				fingerprint, err := id.IdentifyPlatform(conn, asset.Platform, asset.IdDetector)
				if err == nil {
					asset.Name = fingerprint.Name
					asset.PlatformIds = fingerprint.PlatformIDs
					asset.IdDetector = fingerprint.ActiveIdDetectors
				}
			} else {
				// In this case asset.Name should already be set via the inventory
				asset.PlatformIds = []string{pID}
			}

		// Do not expose mock connection as a supported type
		case "mock":
			conn, err = mock.New(connId, "", asset)

		default:
			return nil, errors.New("cannot find connection type " + conf.Type)
		}

		if err != nil {
			return nil, err
		}

		var upstream *upstream.UpstreamClient
		if req.Upstream != nil && !req.Upstream.Incognito {
			upstream, err = req.Upstream.InitClient()
			if err != nil {
				return nil, err
			}
		}

		conf.Id = connId
		conf.Capabilities = conn.Capabilities().String()

		return plugin.NewRuntime(
			conn,
			callback,
			req.HasRecording,
			resources.CreateResource,
			resources.NewResource,
			resources.GetData,
			resources.SetData,
			upstream), nil
	})
	if err != nil {
		return nil, err
	}

	return runtime.Connection.(shared.Connection), nil
}

func (s *Service) discoverRegistry(conn *tar.Connection) (*inventory.Inventory, error) {
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
	// we detect the platform for each asset we discover here
	for _, a := range resolvedAssets {
		// ignore the error. we will retry detection if we connect to the asset
		_ = s.detect(a, conn)
	}
	inventory.AddAssets(resolvedAssets...)

	return inventory, nil
}

func (s *Service) discoverLocalContainers(conf *inventory.Config) (*inventory.Inventory, error) {
	if conf == nil || conf.Discover == nil {
		return nil, nil
	}

	if !stringx.ContainsAnyOf(conf.Discover.Targets, "all", docker_engine.DiscoveryContainerRunning, docker_engine.DiscoveryContainerImages) {
		return nil, nil
	}

	resolvedAssets, err := docker_engine.DiscoverDockerEngineAssets(conf)
	if err != nil {
		return nil, err
	}

	inventory := &inventory.Inventory{}
	inventory.AddAssets(resolvedAssets...)

	return inventory, nil
}

func identifyContainerType(s string) string {
	if strings.Contains(s, ":") || strings.Contains(s, "/") {
		return "docker-image"
	} else {
		return "docker-container"
	}
}
