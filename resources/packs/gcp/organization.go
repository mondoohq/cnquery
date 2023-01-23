package gcp

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"google.golang.org/api/cloudresourcemanager/v1"
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
	(*args)["lifecycleState"] = org.LifecycleState

	return args, nil, nil
}

func (g *mqlGcpOrganization) GetIamPolicy() (interface{}, error) {
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
	policy, err := svc.Organizations.GetIamPolicy(name, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	policyId := fmt.Sprintf("gcp.organization/%s/gcp.iamPolicy", orgId)
	auditConfigs, err := auditConfigsToMql(g.MotorRuntime, policy.AuditConfigs, fmt.Sprintf("%s/auditConfigs", policyId))
	if err != nil {
		return nil, err
	}

	bindings, err := bindingsToMql(g.MotorRuntime, policy.Bindings, fmt.Sprintf("%s/bindings", policyId))
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.iamPolicy",
		"id", policyId,
		"auditConfigs", auditConfigs,
		"bindings", bindings,
		"version", policy.Version,
	)
}

func (g *mqlGcpResourcemanagerBinding) id() (string, error) {
	return g.Id()
}
