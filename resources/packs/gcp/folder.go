package gcp

import (
	"context"
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganizationFoldersService) id() (string, error) {
	id, err := g.OrgId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.organization.foldersService/%s", id), nil
}

func (g *mqlGcpFolder) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.folder/%s", id), nil
}

func (g *mqlGcpOrganizationFoldersService) GetList() ([]interface{}, error) {
	orgId, err := g.OrgId()
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

	folders, err := svc.Folders.List().Parent(orgId).Do()
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

func (g *mqlGcpOrganizationFoldersService) GetAll() ([]interface{}, error) {
	orgId, err := g.OrgId()
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

	filteredFolders := getChildren(folders.Folders, orgId)
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

func (g *mqlGcpFolder) GetFolders() ([]interface{}, error) {
	folderId, err := g.Id()
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

	folders, err := svc.Folders.List().Parent(folderId).Do()
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

func (g *mqlGcpFolder) GetProjects() ([]interface{}, error) {
	folderId, err := g.Id()
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

	projects, err := svc.Projects.List().Parent(folderId).Do()
	if err != nil {
		return nil, err
	}
	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		mqlP, err := g.MotorRuntime.CreateResource("gcp.project",
			"id", p.ProjectId,
			"number", strings.TrimPrefix(p.Name, "projects/")[0:10],
			"name", p.DisplayName,
			"state", p.State,
			"lifecycleState", p.State,
			"createTime", parseTime(p.CreateTime),
			"labels", core.StrMapToInterface(p.Labels),
		)
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlP)
	}
	return mqlProjects, nil
}

func folderToMql(runtime *resources.Runtime, f *cloudresourcemanager.Folder) (interface{}, error) {
	return runtime.CreateResource("gcp.folder",
		"id", f.Name,
		"name", f.DisplayName,
		"created", parseTime(f.CreateTime),
		"updated", parseTime(f.UpdateTime),
		"parent", f.Parent,
		"state", f.State,
	)
}
