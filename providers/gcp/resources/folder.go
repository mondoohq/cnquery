// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpFolders) id() (string, error) {
	if g.ParentId.Error != nil {
		return "", g.ParentId.Error
	}
	id := g.ParentId.Data
	return fmt.Sprintf("gcp.folders/%s", id), nil
}

func (g *mqlGcpFolder) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.folder/%s", id), nil
}

func initGcpFolder(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
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

	folderId := conn.ResourceID()
	if args["id"] != nil {
		folderId = fmt.Sprintf("folders/%s", args["id"].Value.(string))
	}
	folder, err := svc.Folders.Get(folderId).Do()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(folder.Name)
	args["name"] = llx.StringData(folder.DisplayName)
	args["created"] = llx.TimeDataPtr(parseTime(folder.CreateTime))
	args["updated"] = llx.TimeDataPtr(parseTime(folder.CreateTime))
	args["parentId"] = llx.StringData(folder.Parent)
	args["state"] = llx.StringData(folder.State)
	return args, nil, nil
}

func (g *mqlGcpFolders) children() ([]interface{}, error) {
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

	folders, err := svc.Folders.List().Parent(parentId).Do()
	if err != nil {
		return nil, err
	}

	mqlFolders := make([]interface{}, 0, len(folders.Folders))
	for _, f := range folders.Folders {
		mqlF, err := folderToMql(g.MqlRuntime, f)
		if err != nil {
			return nil, err
		}
		mqlFolders = append(mqlFolders, mqlF)
	}
	return mqlFolders, nil
}

func (g *mqlGcpFolders) list() ([]interface{}, error) {
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

	folders, err := svc.Folders.Search().Do()
	if err != nil {
		return nil, err
	}

	filteredFolders := getChildren(folders.Folders, parentId)
	mqlFolders := make([]interface{}, 0, len(filteredFolders))
	for _, f := range filteredFolders {
		mqlF, err := folderToMql(g.MqlRuntime, f)
		if err != nil {
			return nil, err
		}
		mqlFolders = append(mqlFolders, mqlF)
	}
	return mqlFolders, nil
}

func getChildren(fs []*cloudresourcemanager.Folder, root string) []*cloudresourcemanager.Folder {
	var children []*cloudresourcemanager.Folder
	for _, f := range fs {
		if f.Parent == root {
			children = append(children, f)
			children = append(children, getChildren(fs, f.Name)...)
		}
	}
	return children
}

func (g *mqlGcpFolder) folders() (*mqlGcpFolders, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	folderId := g.Id.Data
	res, err := CreateResource(g.MqlRuntime, "gcp.folders", map[string]*llx.RawData{
		"parentId": llx.StringData(folderId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpFolders), nil
}

func (g *mqlGcpFolder) projects() (*mqlGcpProjects, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	folderId := g.Id.Data
	res, err := CreateResource(g.MqlRuntime, "gcp.projects", map[string]*llx.RawData{
		"parentId": llx.StringData(folderId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjects), nil
}

func folderToMql(runtime *plugin.Runtime, f *cloudresourcemanager.Folder) (interface{}, error) {
	return CreateResource(runtime, "gcp.folder", map[string]*llx.RawData{
		"id":       llx.StringData(f.Name),
		"name":     llx.StringData(f.DisplayName),
		"created":  llx.TimeDataPtr(parseTime(f.CreateTime)),
		"updated":  llx.TimeDataPtr(parseTime(f.UpdateTime)),
		"parentId": llx.StringData(f.Parent),
		"state":    llx.StringData(f.State),
	})
}
