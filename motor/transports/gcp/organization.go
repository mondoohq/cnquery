package gcp

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

func (t *Transport) OrganizationID() (string, error) {
	switch t.ResourceType() {
	case Project:
		ctx := context.Background()

		client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
		if err != nil {
			return "", err
		}

		svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return "", err
		}

		ancest, err := svc.Projects.GetAncestry(t.id, &cloudresourcemanager.GetAncestryRequest{}).Do()
		if err != nil {
			return "", err
		}

		for i := range ancest.Ancestor {
			ancestor := ancest.Ancestor[i]
			if strings.ToLower(ancestor.ResourceId.Type) == "organization" {
				return ancestor.ResourceId.Id, nil
			}
		}
	}

	return "", errors.New("could not find the organzation")
}
