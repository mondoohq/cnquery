// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

func (c *mqlCloudflareApiToken) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

// apiTokens lists API tokens visible to the calling user. Cloudflare's API
// tokens are *user* scoped (`/user/tokens`), not account scoped, so they are
// surfaced on the root cloudflare resource — the same set is returned
// regardless of which accounts the user belongs to. Account-owned tokens
// (Enterprise) are not yet exposed by cloudflare-go.
func (c *mqlCloudflare) apiTokens() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	tokens, err := conn.Cf.APITokens(context.TODO())
	if err != nil {
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			return nil, nil
		}
		return nil, err
	}

	var result []any
	for i := range tokens {
		t := tokens[i]

		var ipIn, ipNotIn []any
		if t.Condition != nil && t.Condition.RequestIP != nil {
			ipIn = convert.SliceAnyToInterface(t.Condition.RequestIP.In)
			ipNotIn = convert.SliceAnyToInterface(t.Condition.RequestIP.NotIn)
		}

		policies := make([]any, 0, len(t.Policies))
		for j := range t.Policies {
			p := t.Policies[j]
			pgs := make([]any, 0, len(p.PermissionGroups))
			for k := range p.PermissionGroups {
				pg := p.PermissionGroups[k]
				pgs = append(pgs, map[string]any{
					"id":   pg.ID,
					"name": pg.Name,
				})
			}
			policies = append(policies, map[string]any{
				"id":               p.ID,
				"effect":           p.Effect,
				"resources":        p.Resources,
				"permissionGroups": pgs,
			})
		}

		res, err := CreateResource(c.MqlRuntime, "cloudflare.apiToken", map[string]*llx.RawData{
			"__id":       llx.StringData("cloudflare.apiToken@" + t.ID),
			"id":         llx.StringData(t.ID),
			"name":       llx.StringData(t.Name),
			"status":     llx.StringData(t.Status),
			"issuedOn":   llx.TimeDataPtr(t.IssuedOn),
			"modifiedOn": llx.TimeDataPtr(t.ModifiedOn),
			"notBefore":  llx.TimeDataPtr(t.NotBefore),
			"expiresOn":  llx.TimeDataPtr(t.ExpiresOn),
			"ipIn":       llx.ArrayData(ipIn, types.String),
			"ipNotIn":    llx.ArrayData(ipNotIn, types.String),
			"policies":   llx.ArrayData(policies, types.Dict),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}
