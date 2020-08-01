package gcp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

func (t *Transport) Client(scope ...string) (*http.Client, error) {
	return Client(scope...)
}

func Client(scope ...string) (*http.Client, error) {
	ctx := context.Background()
	return google.DefaultClient(ctx, scope...)
}

func (t *Transport) OrganizationID() (string, error) {
	ctx := context.Background()

	client, err := t.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return "", err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", err
	}

	ancest, err := svc.Projects.GetAncestry(t.projectid, &cloudresourcemanager.GetAncestryRequest{}).Do()
	if err != nil {
		return "", err
	}

	for i := range ancest.Ancestor {
		ancestor := ancest.Ancestor[i]
		if strings.ToLower(ancestor.ResourceId.Type) == "organization" {
			return ancestor.ResourceId.Id, nil
		}
	}
	return "", errors.New("could not find the organzation")
}
