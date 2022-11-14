package google

import (
	"context"
	"errors"
	"strings"

	v1cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (t *Provider) OrganizationID() (string, error) {
	switch t.ResourceType() {
	case Project:
		ctx := context.Background()

		client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
		if err != nil {
			return "", err
		}

		svc, err := v1cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return "", err
		}

		// TODO: GetAncestry is not available in v3 anymore, we need to find an alternative approach
		ancest, err := svc.Projects.GetAncestry(t.id, &v1cloudresourcemanager.GetAncestryRequest{}).Do()
		if err != nil {
			return "", err
		}

		for i := range ancest.Ancestor {
			ancestor := ancest.Ancestor[i]
			if strings.ToLower(ancestor.ResourceId.Type) == "organization" {
				return ancestor.ResourceId.Id, nil
			}
		}
	case Organization:
		return t.id, nil
	}

	return "", errors.New("could not find the organization")
}

func (t *Provider) GetProject(name string) (*cloudresourcemanager.Project, error) {
	ctx := context.Background()

	client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, cloudresourcemanager.CloudPlatformScope, iam.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return svc.Projects.Get("projects/" + name).Do()
}

func (t *Provider) GetOrganization(name string) (*cloudresourcemanager.Organization, error) {
	ctx := context.Background()

	client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, cloudresourcemanager.CloudPlatformScope, iam.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return svc.Organizations.Get("organizations/" + name).Do()
}

func (t *Provider) GetProjectsForOrganization(org *cloudresourcemanager.Organization) ([]*cloudresourcemanager.Project, error) {
	ctx := context.Background()

	client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	projectResp, err := svc.Projects.List().Parent(org.Name).Do()
	if err != nil {
		return nil, err
	}
	return projectResp.Projects, nil
}
