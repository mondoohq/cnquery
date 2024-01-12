// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/gitlab/connection"
	"go.mondoo.com/cnquery/v10/providers/gitlab/resources"
)

const (
	ConnectionType          = "gitlab"
	GitlabGroupConnection   = "gitlab-group"
	GitlabProjectConnection = "gitlab-project"
)

const (
	DiscoveryAuto    = "auto"
	DiscoveryGroup   = "groups"
	DiscoveryProject = "projects"
	// -- chained git discovery options --
	DiscoveryTerraform = "terraform"
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

	token := ""
	if x, ok := flags["token"]; ok && len(x.Value) != 0 {
		token = string(x.Value)
	}
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}
	if token == "" {
		return nil, errors.New("a valid GitLab token is required, pass --token '<yourtoken>' or set GITLAB_TOKEN environment variable")
	}
	conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential("", token))

	if x, ok := flags["group"]; ok && len(x.Value) != 0 {
		conf.Options["group"] = string(x.Value)
	}

	if x, ok := flags["project"]; ok && len(x.Value) != 0 {
		conf.Options["project"] = string(x.Value)
	}
	// it's ok if no group or project is defined.
	// we will discover all the groups
	if x, ok := flags["url"]; ok {
		conf.Options["url"] = string(x.Value)
	}

	conf.Discover = parseDiscover(flags)
	asset := inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

func parseDiscover(flags map[string]*llx.Primitive) *inventory.Discovery {
	var targets []string
	if x, ok := flags["discover"]; ok && len(x.Array) != 0 {
		targets = make([]string, 0, len(x.Array))
		for i := range x.Array {
			entry := string(x.Array[i].Value)
			targets = append(targets, entry)
		}
	} else {
		targets = []string{"auto"}
	}
	return &inventory.Discovery{Targets: targets}
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

	inventory, err := s.discover(req.Asset, conn)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inventory,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.GitLabConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn *connection.GitLabConnection
	var err error

	switch conf.Type {
	default:
		s.lastConnectionID++
		conn, err = connection.NewGitLabConnection(s.lastConnectionID, asset, conf)
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

var (
	projectPlatform = &inventory.Platform{
		Name:    "gitlab-project",
		Title:   "GitLab Project",
		Family:  []string{"gitlab"},
		Kind:    "api",
		Runtime: "gitlab",
	}
	groupPlatform = &inventory.Platform{
		Name:    "gitlab-group",
		Title:   "GitLab Group",
		Family:  []string{"gitlab"},
		Kind:    "api",
		Runtime: "gitlab",
	}
)

func newGitLabGroupID(groupID int) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + strconv.Itoa(groupID)
}

func newGitLabGroupIDFromPath(groupPath string) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + groupPath
}

func newGitLabProjectID(groupID int, projectID int) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + strconv.Itoa(groupID) + "/project/" + strconv.Itoa(projectID)
}

func newGitLabProjectIDFromPaths(groupPath string, projectPath string) string {
	return "//platformid.api.mondoo.app/runtime/gitlab/group/" + groupPath + "/project/" + projectPath
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.GitLabConnection) error {
	asset.Id = conn.Conf.Type

	if !conn.IsGroup() && !conn.IsProject() {
		// that's ok, it means nothing was defined on connection.
		// we will discover the groups
		return nil
	}

	if conn.IsProject() {
		project, err := conn.Project()
		if err != nil {
			return err
		}
		s.detectAsProject(asset, conn.GroupID(), conn.GroupName(), project) // TODO fix 0
	} else {
		group, err := conn.Group()
		if err != nil {
			return err
		}
		s.detectAsGroup(asset, group)
	}
	return nil
}

func (s *Service) detectAsProject(asset *inventory.Asset, groupID int, groupFullPath string, project *gitlab.Project) {
	asset.Platform = projectPlatform
	asset.Name = "GitLab Project " + project.Name
	asset.PlatformIds = []string{
		newGitLabProjectID(groupID, project.ID),
		newGitLabProjectIDFromPaths(groupFullPath, project.Path), // for backwards compatibility with v8
	}
}

func (s *Service) detectAsGroup(asset *inventory.Asset, group *gitlab.Group) error {
	asset.Platform = groupPlatform
	asset.Name = "GitLab Group " + group.Name
	asset.PlatformIds = []string{
		newGitLabGroupID(group.ID),
		newGitLabGroupIDFromPath(group.FullPath), // for backwards compatibility with v8
	}
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
