// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection/gcpinstancesnapshot"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/gcp/resources"
)

const (
	ConnectionType = "gcp"
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

// returns only the env vars that have a set value
func readEnvs(envs ...string) []string {
	vals := []string{}
	for i := range envs {
		val := os.Getenv(envs[i])
		if val != "" {
			vals = append(vals, val)
		}
	}

	return vals
}

// to be used by gcp/googleworkspace cmds, fetches the creds from either the env vars provided or from a flag in the provided cmd
func getGoogleCreds(credentialPath string, envs ...string) []byte {
	var credsPaths []string
	// env vars have precedence over the --credentials-path arg
	credsPaths = readEnvs(envs...)

	if credentialPath != "" {
		credsPaths = append(credsPaths, credentialPath)
	}

	for i := range credsPaths {
		path := credsPaths[i]

		serviceAccount, err := os.ReadFile(path)
		if err == nil {
			return serviceAccount
		}
	}
	return nil
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	if len(req.Args) != 2 {
		return nil, errors.New("missing argument, use `gcp project id`, `gcp organization id`, `gcp folder id`, `gcp instance name`, or `gcp snapshot name`")
	}

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	// custom flag parsing
	var credentialsPath string
	if x, ok := flags["credentials-path"]; ok && len(x.Value) != 0 {
		credentialsPath = string(x.Value)
	}

	// used for snapshot and instance sub-commands
	var projectId string
	if x, ok := flags["project-id"]; ok && len(x.Value) != 0 {
		projectId = string(x.Value)
	}

	var zone string
	if x, ok := flags["zone"]; ok && len(x.Value) != 0 {
		zone = string(x.Value)
	}
	// ^^ snapshot and instance flags

	// these flags are currently only used for the instance sub-command
	var createSnapshot string
	if x, ok := flags["create-snapshot"]; ok && len(x.Value) != 0 {
		createSnapshot = string(x.Value)
	}
	// ^^ instance flags

	envVars := []string{
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GOOGLE_CREDENTIALS",
		"GOOGLE_CLOUD_KEYFILE_JSON",
		"GCLOUD_KEYFILE_JSON",
	}
	serviceAccount := getGoogleCreds(credentialsPath, envVars...)
	if serviceAccount != nil {
		conf.Credentials = append(conf.Credentials, &vault.Credential{
			Type:   vault.CredentialType_json,
			Secret: serviceAccount,
		})
	}

	// parse discovery flags
	conf.Discover = &inventory.Discovery{
		Targets: []string{},
	}
	if x, ok := flags["discover"]; ok && len(x.Array) != 0 {
		for i := range x.Array {
			entry := string(x.Array[i].Value)
			conf.Discover.Targets = append(conf.Discover.Targets, entry)
		}
	} else {
		conf.Discover.Targets = []string{resources.DiscoveryAuto}
	}

	switch req.Args[0] {
	case "org":
		conf.Options["organization-id"] = req.Args[1]
	case "project":
		conf.Options["project-id"] = req.Args[1]
	case "folder":
		conf.Options["folder-id"] = req.Args[1]
	case "gcr":
		conf.Options["project-id"] = req.Args[1]
		conf.Options["repository"] = string(flags["repository"].Value)
		conf.Runtime = "gcp-gcr"
	case "snapshot":
		conf.Options["snapshot-name"] = req.Args[1]
		conf.Options["project-id"] = projectId
		conf.Options["zone"] = zone
		conf.Options["type"] = "snapshot"
		conf.Type = string(gcpinstancesnapshot.SnapshotConnectionType)
		conf.Discover = nil
	case "instance":
		conf.Options["instance-name"] = req.Args[1]
		conf.Options["type"] = "instance"
		conf.Options["project-id"] = projectId
		conf.Options["zone"] = zone
		conf.Options["create-snapshot"] = createSnapshot
		conf.Type = string(gcpinstancesnapshot.SnapshotConnectionType)
		conf.Discover = nil
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	for i := range s.runtimes {
		runtime := s.runtimes[i]
		// FIXME: I think, we might need the asset here to cleanup the correct connection
		sharedConn := runtime.Connection.(shared.GcpConnection)
		if sharedConn.Type() == gcpinstancesnapshot.SnapshotConnectionType {
			conn := runtime.Connection.(*gcpinstancesnapshot.GcpSnapshotConnection)
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

	var inventory *inventory.Inventory
	// discovery assets for further scanning
	if conn.Config().Discover != nil {
		// detection of the platform is done in the discovery phase
		inventory, err = s.discover(conn)
		if err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inventory,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.GcpConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn shared.GcpConnection
	var err error

	switch conf.Type {
	case string(gcpinstancesnapshot.SnapshotConnectionType):
		// A GcpSnapshotConnection is a wrapper around a FilesystemConnection
		// To make sure the connection is later handled by the os provider, override the type
		conf.Type = "filesystem"
		s.lastConnectionID++
		conn, err = gcpinstancesnapshot.NewGcpSnapshotConnection(s.lastConnectionID, conf, asset)
	default:
		s.lastConnectionID++
		conn, err = connection.NewGcpConnection(s.lastConnectionID, asset, conf)
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

func (s *Service) discover(conn shared.GcpConnection) (*inventory.Inventory, error) {
	if conn.Config().Discover == nil {
		return nil, nil
	}

	runtime, ok := s.runtimes[conn.ID()]
	if !ok {
		// no connection found, this should never happen
		return nil, errors.New("connection " + strconv.FormatUint(uint64(conn.ID()), 10) + " not found")
	}

	return resources.Discover(runtime)
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
