// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/network/connection"
	"go.mondoo.com/cnquery/v10/providers/network/resources"
	"go.mondoo.com/cnquery/v10/providers/network/resources/domain"
)

const (
	defaultConnection  uint32 = 1
	HostConnectionType        = "host"
)

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
	target := req.Args[0]

	host, port, scheme, path, err := parseTarget(target)
	if err != nil {
		return nil, err
	}

	insecure := false
	if found, ok := req.Flags["insecure"]; ok {
		insecure, _ = found.RawData().Value.(bool)
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{{
			Type:     "host",
			Port:     int32(port),
			Host:     host,
			Path:     path,
			Runtime:  scheme,
			Insecure: insecure,
		}},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

func parseTarget(target string) (string, int, string, string, error) {
	// Note on noSchema handling:
	// A user may type in a target like: `google.com`. Technically, this is not
	// a valid scheme. We need to make it into a valid url scheme for parsing
	// and further processing, but we also want to be mindful of what users intend.
	//
	// If we set this to e.g. an HTTP scheme with port 80, then we break
	// the assumptions that users have when they use other resources (like TLS).
	// For example: `host google.com` and command `tls.versions` is a user
	// indicating that they want the TLS config of https://google.com of course.
	// However, we also want to use HTTP:80 when we do `http.get` requests,
	// because that is the default way this is handled in the web (for now).
	//
	// Thus: the scheme becomes empty "" and the port is set to 0. Every resource
	// needs to figure out what that means to it.
	noScheme := false
	if i := strings.Index(target, "://"); i == -1 {
		noScheme = true
		target = "http://" + target
	}

	url, err := url.Parse(target)
	if err != nil {
		return "", 0, "", "", err
	}

	host, port := domain.SplitHostPort(url.Host)
	if port == 0 && !noScheme {
		port = resources.CommonPorts[url.Scheme]
	}

	scheme := url.Scheme
	if noScheme {
		scheme = ""
	}

	path := url.Path

	return host, port, scheme, path, nil
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
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

	// TODO: discovery of related assets and use them in the inventory below

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.HostConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn *connection.HostConnection
	var err error

	switch conf.Type {
	case HostConnectionType:
		s.lastConnectionID++
		conn = connection.NewHostConnection(s.lastConnectionID, asset, conf)

	default:
		// generic host connection, without anything else
		s.lastConnectionID++
		conn = connection.NewHostConnection(s.lastConnectionID, asset, conf)
	}

	if conn.Conf.Options != nil && conn.Conf.Options["host"] != "" {
		target := conn.Conf.Options["host"]
		host, port, scheme, path, err := parseTarget(target)
		if err != nil {
			return nil, err
		}
		conn.Conf.Host = host
		conn.Conf.Path = path
		conn.Conf.Port = int32(port)
		conn.Conf.Runtime = scheme
	}

	if err != nil {
		return nil, err
	}

	conf.Backend = inventory.ProviderType_HOST
	conf.Kind = inventory.DeprecatedV8_Kind_KIND_NETWORK

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

func (s *Service) detect(asset *inventory.Asset, conn *connection.HostConnection) error {
	if conn.Conf.Runtime != "" {
		hostWithScheme := conn.Conf.Runtime + "://" + conn.Conf.Host
		asset.Name = hostWithScheme
	} else {
		asset.Name = conn.Conf.Host
	}
	asset.Platform = &inventory.Platform{
		Name:   "host",
		Family: []string{"network"},
		Kind:   "network",
		Title:  "Network Host",
	}

	asset.Fqdn = conn.FQDN()
	asset.PlatformIds = []string{"//platformid.api.mondoo.app/runtime/network/host/" + conn.Conf.Runtime + conn.Conf.Host}

	return nil
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
