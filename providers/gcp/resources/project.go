// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjects) id() (string, error) {
	if g.ParentId.Error != nil {
		return "", g.ParentId.Error
	}
	id := g.ParentId.Data
	return fmt.Sprintf("gcp.projects/%s", id), nil
}

func initGcpProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args == nil {
		args = make(map[string]*llx.RawData)
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	projectId := fmt.Sprintf("projects/%s", conn.ResourceID())
	project, err := svc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(project.ProjectId)
	args["number"] = llx.StringData(strings.TrimPrefix(project.Name, "projects/")[0:10])
	args["name"] = llx.StringData(project.DisplayName)
	args["parentId"] = llx.StringData(project.Parent)
	args["state"] = llx.StringData(project.State)
	args["lifecycleState"] = llx.StringData(project.State)
	args["createTime"] = llx.TimeDataPtr(parseTime(project.CreateTime))
	args["labels"] = llx.MapData(convert.MapToInterfaceMap(project.Labels), types.String)
	// TODO: add organization gcp.organization
	return args, nil, nil
}

func (g *mqlGcpProject) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProject) name() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) parentId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) number() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) state() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) lifecycleState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) createTime() (*time.Time, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) labels() (map[string]interface{}, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) iamPolicy() ([]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := svc.Projects.GetIamPolicy(fmt.Sprintf("projects/%s", projectId), &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":      llx.StringData(projectId + "-" + strconv.Itoa(i)),
			"role":    llx.StringData(b.Role),
			"members": llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpProject) commonInstanceMetadata() (map[string]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	p, err := computeSvc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	if p.CommonInstanceMetadata != nil {
		for _, item := range p.CommonInstanceMetadata.Items {
			value := ""
			if item.Value != nil {
				value = *item.Value
			}
			metadata[item.Key] = value
		}
	}
	return convert.MapToInterfaceMap(metadata), nil
}

func (g *mqlGcpProjects) children() ([]interface{}, error) {
	if g.ParentId.Error != nil {
		return nil, g.ParentId.Error
	}
	parentId := g.ParentId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projects, err := svc.Projects.List().Parent(parentId).Do()
	if err != nil {
		return nil, err
	}

	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		mqlP, err := projectToMql(g.MqlRuntime, p)
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlP)
	}
	return mqlProjects, nil
}

func (g *mqlGcpProjects) list() ([]interface{}, error) {
	if g.ParentId.Error != nil {
		return nil, g.ParentId.Error
	}
	parentId := g.ParentId.Data

	obj, err := CreateResource(g.MqlRuntime, "gcp.folders", map[string]*llx.RawData{
		"parentId": llx.StringData(parentId),
	})
	if err != nil {
		return nil, err
	}
	foldersSvc := obj.(*mqlGcpFolders)
	folders := foldersSvc.GetList()
	if folders.Error != nil {
		return nil, folders.Error
	}

	foldersMap := map[string]struct{}{parentId: {}}
	for _, f := range folders.Data {
		id := f.(*mqlGcpFolder).GetId()
		if id.Error != nil {
			return nil, id.Error
		}
		foldersMap[id.Data] = struct{}{}
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projects, err := svc.Projects.Search().Do()
	if err != nil {
		return nil, err
	}
	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		if _, ok := foldersMap[p.Parent]; ok {
			mqlP, err := projectToMql(g.MqlRuntime, p)
			if err != nil {
				return nil, err
			}
			mqlProjects = append(mqlProjects, mqlP)
		}
	}
	return mqlProjects, nil
}

func projectToMql(runtime *plugin.Runtime, p *cloudresourcemanager.Project) (*mqlGcpProject, error) {
	res, err := CreateResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id":         llx.StringData(p.ProjectId),
		"number":     llx.StringData(strings.TrimPrefix(p.Name, "projects/")[0:10]),
		"name":       llx.StringData(p.DisplayName),
		"parentId":   llx.StringData(p.Parent),
		"state":      llx.StringData(p.State),
		"createTime": llx.TimeDataPtr(parseTime(p.CreateTime)),
		"labels":     llx.MapData(convert.MapToInterfaceMap(p.Labels), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProject), nil
}
