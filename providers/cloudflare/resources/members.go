// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

func (c *mqlCloudflareAccountMember) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return "cloudflare/account/member/" + c.Id.Data, nil
}

func (c *mqlCloudflareAccount) members() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	ctx := context.TODO()

	results := []any{}
	page := 1
	for {
		members, info, err := conn.Cf.AccountMembers(ctx, c.Id.Data, cloudflare.PaginationOptions{Page: page, PerPage: 50})
		if err != nil {
			return nil, err
		}
		for i := range members {
			m := members[i]

			roles := make([]any, 0, len(m.Roles))
			for j := range m.Roles {
				r := m.Roles[j]
				role, err := NewResource(c.MqlRuntime, "cloudflare.account.role", map[string]*llx.RawData{
					"id":          llx.StringData(r.ID),
					"name":        llx.StringData(r.Name),
					"description": llx.StringData(r.Description),
				})
				if err != nil {
					return nil, err
				}
				roles = append(roles, role)
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.account.member", map[string]*llx.RawData{
				"id":                             llx.StringData(m.ID),
				"status":                         llx.StringData(m.Status),
				"userId":                         llx.StringData(m.User.ID),
				"email":                          llx.StringData(m.User.Email),
				"firstName":                      llx.StringData(m.User.FirstName),
				"lastName":                       llx.StringData(m.User.LastName),
				"twoFactorAuthenticationEnabled": llx.BoolData(m.User.TwoFactorAuthenticationEnabled),
				"roles":                          llx.ArrayData(roles, types.Resource("cloudflare.account.role")),
			})
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}

		if info.Page >= info.TotalPages || len(members) == 0 {
			break
		}
		page++
	}
	return results, nil
}
