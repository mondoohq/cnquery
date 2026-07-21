// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlMongodbatlas) resourcePolicies() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	policies, httpResp, err := atlasClient(r.MqlRuntime).ResourcePoliciesApi.ListOrgResourcePolicies(context.Background(), oid).Execute()
	if err != nil {
		// Resource policies are a feature-gated org capability; a credential
		// without access or an org without the feature degrades to an empty
		// list rather than failing the scan, matching the singleton settings.
		if isAccessDenied(httpResp) || (httpResp != nil && httpResp.StatusCode == http.StatusNotFound) {
			return nil, nil
		}
		return nil, err
	}

	out := []any{}
	for i := range policies {
		p := policies[i]

		statements := []any{}
		for _, stmt := range p.GetPolicies() {
			statements = append(statements, map[string]any{
				"id":   stmt.GetId(),
				"body": stmt.GetBody(),
			})
		}

		createdBy := p.GetCreatedByUser()
		lastUpdatedBy := p.GetLastUpdatedByUser()

		res, err := CreateResource(r.MqlRuntime, "mongodbatlas.resourcePolicy", map[string]*llx.RawData{
			"__id":              llx.StringData("mongodbatlas.resourcePolicy/" + oid + "/" + p.GetId()),
			"id":                llx.StringData(p.GetId()),
			"name":              llx.StringData(p.GetName()),
			"description":       llx.StringData(p.GetDescription()),
			"createdByUser":     llx.StringData(createdBy.GetName()),
			"createdDate":       llx.TimeDataPtr(timePtr(p.GetCreatedDate())),
			"lastUpdatedByUser": llx.StringData(lastUpdatedBy.GetName()),
			"lastUpdatedDate":   llx.TimeDataPtr(timePtr(p.GetLastUpdatedDate())),
			"policies":          llx.ArrayData(statements, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
