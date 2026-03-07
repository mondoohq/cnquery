// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
)

func (g *mqlGcpProjectIamService) roles() ([]any, error) {
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

	var roles []any
	it := adminSvc.ListRolesIter(ctx, &adminpb.ListRolesRequest{
		Parent:      fmt.Sprintf("projects/%s", projectId),
		View:        adminpb.RoleView_FULL,
		ShowDeleted: true,
	})
	for {
		r, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		permissions := make([]any, 0, len(r.IncludedPermissions))
		for _, p := range r.IncludedPermissions {
			permissions = append(permissions, p)
		}

		mqlRole, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.role", map[string]*llx.RawData{
			"projectId":           llx.StringData(projectId),
			"name":                llx.StringData(r.Name),
			"title":               llx.StringData(r.Title),
			"description":         llx.StringData(r.Description),
			"stage":               llx.StringData(r.Stage.String()),
			"includedPermissions": llx.ArrayData(permissions, types.String),
			"deleted":             llx.BoolData(r.Deleted),
		})
		if err != nil {
			return nil, err
		}
		roles = append(roles, mqlRole)
	}
	return roles, nil
}

func (g *mqlGcpProjectIamServiceRole) id() (string, error) {
	return g.Name.Data, g.Name.Error
}
