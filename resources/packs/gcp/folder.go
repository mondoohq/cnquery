package gcp

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpFolders) id() (string, error) {
	id, err := g.ParentId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.folders/%s", id), nil
}

func (g *mqlGcpFolder) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.folder/%s", id), nil
}

func (g *mqlGcpFolder) init(args *resources.Args) (*resources.Args, GcpFolder, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	folderId := provider.ResourceID()
	folder, err := svc.Folders.Get(folderId).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = folder.Name
	(*args)["name"] = folder.DisplayName
	(*args)["created"] = parseTime(folder.CreateTime)
	(*args)["updated"] = parseTime(folder.CreateTime)
	(*args)["parentId"] = folder.Parent
	(*args)["state"] = folder.State
	return args, nil, nil
}

func (g *mqlGcpFolders) GetChildren() ([]interface{}, error) {
	parentId, err := g.ParentId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
		mqlF, err := folderToMql(g.MotorRuntime, f)
		if err != nil {
			return nil, err
		}
		mqlFolders = append(mqlFolders, mqlF)
	}
	return mqlFolders, nil
}

func (g *mqlGcpFolders) GetList() ([]interface{}, error) {
	parentId, err := g.ParentId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
		mqlF, err := folderToMql(g.MotorRuntime, f)
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

func (g *mqlGcpFolder) GetFolders() (interface{}, error) {
	folderId, err := g.Id()
	if err != nil {
		return nil, err
	}
	return g.MotorRuntime.CreateResource("gcp.folders", "parentId", folderId)
}

func (g *mqlGcpFolder) GetProjects() (interface{}, error) {
	folderId, err := g.Id()
	if err != nil {
		return nil, err
	}
	return g.MotorRuntime.CreateResource("gcp.projects", "parentId", folderId)
}

func folderToMql(runtime *resources.Runtime, f *cloudresourcemanager.Folder) (interface{}, error) {
	return runtime.CreateResource("gcp.folder",
		"id", f.Name,
		"name", f.DisplayName,
		"created", parseTime(f.CreateTime),
		"updated", parseTime(f.UpdateTime),
		"parentId", f.Parent,
		"state", f.State,
	)
}
