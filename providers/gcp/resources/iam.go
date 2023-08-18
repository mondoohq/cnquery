package resources

import (
	"context"
	"errors"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
)

func (g *mqlGcpProjectIamService) id() (string, error) {
	if err := g.ProjectId.Error; err != nil {
		return "", err
	}
	return g.ProjectId.Data + "gcp.project.iamService", nil
}

func (g *mqlGcpProject) iam() (*mqlGcpProjectIamService, error) {
	if err := g.Id.Error; err != nil {
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamService), nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) id() (string, error) {
	return g.UniqueId.Data, nil
}

func (g *mqlGcpProjectIamServiceServiceAccountKey) id() (string, error) {
	return g.Name.Data, nil
}

func initGcpProjectIamService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	iamSvc := obj.(*mqlGcpProjectIamService)
	sas := iamSvc.GetServiceAccounts()
	if sas.Error != nil {
		return nil, nil, sas.Error
	}

	for _, s := range sas.Data {
		sa := s.(*mqlGcpProjectIamServiceServiceAccount)
		email := sa.GetEmail()
		if email.Error != nil {
			return nil, nil, email.Error
		}

		if email.Data == args["email"].Value.(string) {
			return args, sa, nil
		}
	}
	return nil, nil, errors.New("service account not found")
}

func (g *mqlGcpProjectIamService) serviceAccounts() ([]interface{}, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	var serviceAccounts []interface{}
	it := adminSvc.ListServiceAccounts(ctx, &adminpb.ListServiceAccountsRequest{Name: fmt.Sprintf("projects/%s", g.ProjectId.Data)})
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlSA, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
			"projectId":      llx.StringData(s.ProjectId),
			"name":           llx.StringData(s.Name),
			"uniqueId":       llx.StringData(s.UniqueId),
			"email":          llx.StringData(s.Email),
			"displayName":    llx.StringData(s.DisplayName),
			"description":    llx.StringData(s.Description),
			"oauth2ClientId": llx.StringData(s.Oauth2ClientId),
			"disabled":       llx.BoolData(s.Disabled),
		})
		if err != nil {
			return nil, err
		}
		serviceAccounts = append(serviceAccounts, mqlSA)
	}
	return serviceAccounts, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) keys() ([]interface{}, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}

	if err := g.Email.Error; err != nil {
		return nil, err
	}

	projectId := g.ProjectId.Data
	email := g.Email.Data

	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	resp, err := adminSvc.ListServiceAccountKeys(ctx, &adminpb.ListServiceAccountKeysRequest{Name: fmt.Sprintf("projects/%s/serviceAccounts/%s", projectId, email)})
	if err != nil {
		return nil, err
	}
	mqlKeys := make([]interface{}, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		mqlKey, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount.key", map[string]*llx.RawData{
			"name":            llx.StringData(k.Name),
			"keyAlgorithm":    llx.StringData(k.KeyAlgorithm.String()),
			"validAfterTime":  llx.TimeDataPtr(timestampAsTimePtr(k.ValidAfterTime)),
			"validBeforeTime": llx.TimeDataPtr(timestampAsTimePtr(k.ValidBeforeTime)),
			"keyOrigin":       llx.StringData(k.KeyOrigin.String()),
			"keyType":         llx.StringData(k.KeyType.String()),
			"disabled":        llx.BoolData(k.Disabled),
		})
		if err != nil {
			return nil, err
		}
		mqlKeys = append(mqlKeys, mqlKey)
	}
	return mqlKeys, nil
}
