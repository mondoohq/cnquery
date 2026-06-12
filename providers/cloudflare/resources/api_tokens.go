// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"

	"github.com/cloudflare/cloudflare-go/v6/user"
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

	var result []any
	iter := conn.Cf.User.Tokens.ListAutoPaging(context.TODO(), user.TokenListParams{})
	for iter.Next() {
		t := iter.Current()

		ipIn := convert.SliceAnyToInterface(t.Condition.RequestIP.In)
		ipNotIn := convert.SliceAnyToInterface(t.Condition.RequestIP.NotIn)

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
			// The resources field is a polymorphic union in cloudflare-go v6;
			// round-trip it through JSON so the dict holds a plain decoded value
			// matching the raw API shape.
			var resources any
			if b, err := json.Marshal(p.Resources); err == nil {
				_ = json.Unmarshal(b, &resources)
			}
			policies = append(policies, map[string]any{
				"id":               p.ID,
				"effect":           string(p.Effect),
				"resources":        resources,
				"permissionGroups": pgs,
			})
		}

		res, err := CreateResource(c.MqlRuntime, "cloudflare.apiToken", map[string]*llx.RawData{
			"__id":       llx.StringData("cloudflare.apiToken@" + t.ID),
			"id":         llx.StringData(t.ID),
			"name":       llx.StringData(t.Name),
			"status":     llx.StringData(string(t.Status)),
			"issuedOn":   timeOrNil(t.IssuedOn),
			"modifiedOn": timeOrNil(t.ModifiedOn),
			"notBefore":  timeOrNil(t.NotBefore),
			"expiresOn":  timeOrNil(t.ExpiresOn),
			"ipIn":       llx.ArrayData(ipIn, types.String),
			"ipNotIn":    llx.ArrayData(ipNotIn, types.String),
			"policies":   llx.ArrayData(policies, types.Dict),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		// Token listing requires a token with the right permissions; treat
		// permission/availability errors as an empty result rather than failing.
		if isUnavailable(err) {
			return nil, nil
		}
		return nil, err
	}

	return result, nil
}
