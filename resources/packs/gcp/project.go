package gcp

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) id() (string, error) {
	return "gcp.project", nil
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

	projectId := provider.ResourceID()
	project, err := svc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = project.ProjectId
	(*args)["number"] = strconv.FormatInt(project.ProjectNumber, 10)
	(*args)["name"] = project.Name
	(*args)["state"] = project.LifecycleState
	(*args)["lifecycleState"] = project.LifecycleState
	var createTime *time.Time
	parsedTime, err := time.Parse(time.RFC3339, project.CreateTime)
	if err != nil {
		return nil, nil, errors.New("could not parse gcp.project create time: " + project.CreateTime)
	} else {
		createTime = &parsedTime
	}
	(*args)["createTime"] = createTime
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
