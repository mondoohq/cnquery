package gcp

import (
	"context"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganization) id() (string, error) {
	return "gcp.organization", nil
}

func (g *mqlGcpOrganization) init(args *resources.Args) (*resources.Args, GcpOrganization, error) {
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

	// determine org from project in transport
	orgId, err := provider.OrganizationID()
	if err != nil {
		log.Error().Err(err).Msg("could not determine organization id")
		return nil, nil, err
	}

	name := "organizations/" + orgId
	org, err := svc.Organizations.Get(name).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = org.Name
	(*args)["name"] = org.DisplayName
	(*args)["state"] = org.State
	(*args)["lifecycleState"] = org.State

	return args, nil, nil
}

func (g *mqlGcpOrganization) GetIamPolicy() ([]interface{}, error) {
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

	// determine org from project in transport
	orgId, err := provider.OrganizationID()
	if err != nil {
		return nil, err
	}

	name := "organizations/" + orgId
	orgpolicy, err := svc.Organizations.GetIamPolicy(name, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range orgpolicy.Bindings {
		b := orgpolicy.Bindings[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.resourcemanager.binding",
			"id", name+"-"+strconv.Itoa(i),
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

func (g *mqlGcpOrganization) GetProjects() ([]interface{}, error) {
	orgId, err := g.Id()
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

	projects, err := svc.Projects.List().Parent(orgId).Do()
	if err != nil {
		return nil, err
	}

	mqlProjects := make([]interface{}, 0, len(projects.Projects))
	for _, p := range projects.Projects {
		project, err := g.MotorRuntime.CreateResource("gcp.project",
			"id", p.ProjectId,
			"number", strings.TrimPrefix(p.Name, "projects/")[0:10],
			"name", p.DisplayName,
			"state", p.State,
			"createTime", parseTime(p.CreateTime),
			"labels", core.StrMapToInterface(p.Labels),
		)
		if err != nil {
			return nil, err
		}
		mqlProjects = append(mqlProjects, project)
	}
	return mqlProjects, nil
}

func (g *mqlGcpResourcemanagerBinding) id() (string, error) {
	return g.Id()
}
