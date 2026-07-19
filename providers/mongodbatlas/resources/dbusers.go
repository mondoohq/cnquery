// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlMongodbatlas) databaseUsers() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.DatabaseUsersApi.ListDatabaseUsers(ctx, pid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			u := results[i]

			roles := []any{}
			for _, role := range u.GetRoles() {
				roles = append(roles, map[string]any{
					"roleName":       role.GetRoleName(),
					"databaseName":   role.GetDatabaseName(),
					"collectionName": role.GetCollectionName(),
				})
			}
			scopes := []any{}
			for _, sc := range u.GetScopes() {
				scopes = append(scopes, map[string]any{
					"name": sc.GetName(),
					"type": sc.GetType(),
				})
			}

			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.databaseUser", map[string]*llx.RawData{
				"__id":            llx.StringData("mongodbatlas.databaseUser/" + pid + "/" + u.GetDatabaseName() + "/" + u.GetUsername()),
				"id":              llx.StringData(u.GetDatabaseName() + "/" + u.GetUsername()),
				"username":        llx.StringData(u.GetUsername()),
				"databaseName":    llx.StringData(u.GetDatabaseName()),
				"description":     llx.StringData(u.GetDescription()),
				"awsIAMType":      llx.StringData(u.GetAwsIAMType()),
				"x509Type":        llx.StringData(u.GetX509Type()),
				"ldapAuthType":    llx.StringData(u.GetLdapAuthType()),
				"oidcAuthType":    llx.StringData(u.GetOidcAuthType()),
				"roles":           llx.ArrayData(roles, types.Dict),
				"scopes":          llx.ArrayData(scopes, types.Dict),
				"deleteAfterDate": llx.TimeDataPtr(timePtr(u.GetDeleteAfterDate())),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}
