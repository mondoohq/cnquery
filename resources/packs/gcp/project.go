package gcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjects) id() (string, error) {
	id, err := g.ParentId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.projects/%s", id), nil
}

func (g *mqlGcpProject) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project/%s", id), nil
}

func (g *mqlGcpProject) init(args *resources.Args) (*resources.Args, GcpProject, error) {
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

	projectId := fmt.Sprintf("projects/%s", provider.ResourceID())
	project, err := svc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = project.ProjectId
	(*args)["number"] = strings.TrimPrefix(project.Name, "projects/")[0:10]
	(*args)["name"] = project.Name
	(*args)["parentId"] = project.Parent
	(*args)["state"] = project.State
	(*args)["lifecycleState"] = project.State
	(*args)["createTime"] = parseTime(project.CreateTime)
	(*args)["labels"] = core.StrMapToInterface(project.Labels)
	// TODO: add organization gcp.organization
	return args, nil, nil
}

func (g *mqlGcpProject) GetId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetName() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetParentId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetNumber() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetLifecycleState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetCreateTime() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) GetLabels() (map[string]interface{}, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) GetIamPolicy() ([]interface{}, error) {
	projectId, err := g.Id()
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

	policy, err := svc.Projects.GetIamPolicy(projectId, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.resourcemanager.binding",
			"id", projectId+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpProject) GetCommonInstanceMetadata() (map[string]interface{}, error) {
	projectId, err := g.Id()
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
			metadata[item.Key] = core.ToString(item.Value)
		}
	}
	return core.StrMapToInterface(metadata), nil
}

func (g *mqlGcpProjects) GetChildren() ([]interface{}, error) {
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

	projects, err := svc.Projects.List().Parent(parentId).Do()
	if err != nil {
		return nil, err
	}

	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		mqlP, err := projectToMql(g.MotorRuntime, p)
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, mqlP)
	}
	return mqlProjects, nil
}

func (g *mqlGcpProjects) GetList() ([]interface{}, error) {
	parentId, err := g.ParentId()
	if err != nil {
		return nil, err
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.folders", "parentId", parentId)
	if err != nil {
		return nil, err
	}
	foldersSvc := obj.(GcpFolders)
	folders, err := foldersSvc.List()
	if err != nil {
		return nil, err
	}

	foldersMap := map[string]struct{}{parentId: {}}
	for _, f := range folders {
		id, err := f.(GcpFolder).Id()
		if err != nil {
			return nil, err
		}
		foldersMap[id] = struct{}{}
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

	projects, err := svc.Projects.Search().Do()
	if err != nil {
		return nil, err
	}
	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		if _, ok := foldersMap[p.Parent]; ok {
			mqlP, err := projectToMql(g.MotorRuntime, p)
			if err != nil {
				return nil, err
			}
			mqlProjects = append(mqlProjects, mqlP)
		}
	}
	return mqlProjects, nil
}

func projectToMql(runtime *resources.Runtime, p *cloudresourcemanager.Project) (interface{}, error) {
	return runtime.CreateResource("gcp.project",
		"id", p.ProjectId,
		"number", strings.TrimPrefix(p.Name, "projects/")[0:10],
		"name", p.DisplayName,
		"parentId", p.Parent,
		"state", p.State,
		"createTime", parseTime(p.CreateTime),
		"labels", core.StrMapToInterface(p.Labels),
	)
}
