package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// func (g *mqlGcpProjects) id() (string, error) {
// 	id, err := g.ParentId()
// 	if err != nil {
// 		return "", err
// 	}
// 	return fmt.Sprintf("gcp.projects/%s", id), nil
// }

func (g *mqlGcpProject) id() (string, error) {
	return "gcp.project/" + g.Id.Data, nil
}

func initGcpProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(runtime.Connection)
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

	args["id"] = llx.StringData(project.ProjectId)
	args["number"] = llx.StringData(strings.TrimPrefix(project.Name, "projects/"))
	args["name"] = llx.StringData(project.DisplayName)
	args["parentId"] = llx.StringData(project.Parent)
	args["state"] = llx.StringData(project.State)
	args["lifecycleState"] = llx.StringData(project.State)
	if cTime := parseTime(project.CreateTime); cTime != nil {
		args["createTime"] = llx.TimeData(*cTime)
	}

	args["labels"] = llx.MapData(convert.MapToInterfaceMap(project.Labels), types.String)
	// TODO: add organization gcp.organization
	return args, nil, nil
}

func (g *mqlGcpProject) iamPolicy() ([]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}

	provider, err := gcpProvider(g.MqlRuntime.Connection)
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

	projectId := g.Id.Data
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

// func (g *mqlGcpProject) GetCommonInstanceMetadata() (map[string]interface{}, error) {
// 	projectId, err := g.Id()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	p, err := computeSvc.Projects.Get(projectId).Do()
// 	if err != nil {
// 		return nil, err
// 	}

// 	metadata := make(map[string]string)
// 	if p.CommonInstanceMetadata != nil {
// 		for _, item := range p.CommonInstanceMetadata.Items {
// 			metadata[item.Key] = core.ToString(item.Value)
// 		}
// 	}
// 	return core.StrMapToInterface(metadata), nil
// }

// func (g *mqlGcpProjects) GetChildren() ([]interface{}, error) {
// 	parentId, err := g.ParentId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	projects, err := svc.Projects.List().Parent(parentId).Do()
// 	if err != nil {
// 		return nil, err
// 	}

// 	mqlProjects := make([]interface{}, 0, len(projects.Projects))
// 	for _, p := range projects.Projects {
// 		mqlP, err := projectToMql(g.MotorRuntime, p)
// 		if err != nil {
// 			return nil, err
// 		}
// 		mqlProjects = append(mqlProjects, mqlP)
// 	}
// 	return mqlProjects, nil
// }

// func (g *mqlGcpProjects) GetList() ([]interface{}, error) {
// 	parentId, err := g.ParentId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	obj, err := g.MotorRuntime.CreateResource("gcp.folders", "parentId", parentId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	foldersSvc := obj.(GcpFolders)
// 	folders, err := foldersSvc.List()
// 	if err != nil {
// 		return nil, err
// 	}

// 	foldersMap := map[string]struct{}{parentId: {}}
// 	for _, f := range folders {
// 		id, err := f.(GcpFolder).Id()
// 		if err != nil {
// 			return nil, err
// 		}
// 		foldersMap[id] = struct{}{}
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	projects, err := svc.Projects.Search().Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	mqlProjects := make([]interface{}, 0, len(projects.Projects))
// 	for _, p := range projects.Projects {
// 		if _, ok := foldersMap[p.Parent]; ok {
// 			mqlP, err := projectToMql(g.MotorRuntime, p)
// 			if err != nil {
// 				return nil, err
// 			}
// 			mqlProjects = append(mqlProjects, mqlP)
// 		}
// 	}
// 	return mqlProjects, nil
// }

// func projectToMql(runtime *resources.Runtime, p *cloudresourcemanager.Project) (interface{}, error) {
// 	return runtime.CreateResource("gcp.project",
// 		"id", p.ProjectId,
// 		"number", strings.TrimPrefix(p.Name, "projects/")[0:10],
// 		"name", p.DisplayName,
// 		"parentId", p.Parent,
// 		"state", p.State,
// 		"createTime", parseTime(p.CreateTime),
// 		"labels", core.StrMapToInterface(p.Labels),
// 	)
// }
