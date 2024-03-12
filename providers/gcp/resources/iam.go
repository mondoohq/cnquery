// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
)

func (g *mqlGcpProjectIamService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.iamService", projectId), nil
}

func (g *mqlGcpProject) iam() (*mqlGcpProjectIamService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	return res.(*mqlGcpProjectIamService), err
}

func (g *mqlGcpProjectIamServiceServiceAccount) id() (string, error) {
	return g.UniqueId.Data, g.UniqueId.Error
}

func (g *mqlGcpProjectIamServiceServiceAccountKey) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func initGcpProjectIamServiceServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
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

		if email.Data == args["email"].Value {
			return args, sa, nil
		}
	}
	return nil, nil, errors.New("service account not found")
}

func (g *mqlGcpProjectIamService) serviceAccounts() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(admin.DefaultAuthScopes()...)
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
	it := adminSvc.ListServiceAccounts(ctx, &adminpb.ListServiceAccountsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
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
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Email.Error != nil {
		return nil, g.Email.Error
	}
	email := g.Email.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(admin.DefaultAuthScopes()...)
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
