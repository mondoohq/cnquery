// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/netlify/connection"
	"go.mondoo.com/mql/v13/types"
)

type envVarRecord struct {
	Key       string            `json:"key"`
	Scopes    []string          `json:"scopes"`
	IsSecret  bool              `json:"is_secret"`
	UpdatedAt netlifyTime       `json:"updated_at"`
	UpdatedBy *envVarUserData   `json:"updated_by"`
	Values    []envVarValueData `json:"values"`
}

type envVarUserData struct {
	Email string `json:"email"`
}

type envVarValueData struct {
	Context          string `json:"context"`
	ContextParameter string `json:"context_parameter"`
	Value            string `json:"value"`
}

func (a *mqlNetlifyAccount) environmentVariables() ([]any, error) {
	return fetchEnvVars(a.MqlRuntime, a.Id.Data, "", nil)
}

// fetchEnvVars reads the environment variables of an account, optionally
// narrowed to one site, and builds a resource per variable. The owner
// identifiers form the cache key so an account-wide variable and the same
// variable read through a site stay distinct.
func fetchEnvVars(runtime *plugin.Runtime, accountID, siteID string, query url.Values) ([]any, error) {
	c := netlifyConn(runtime)

	records, err := connection.GetList[envVarRecord](context.Background(), c,
		"/accounts/"+url.PathEscape(accountID)+"/env", query)
	if err != nil {
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]

		values := make([]any, 0, len(rec.Values))
		for _, v := range rec.Values {
			values = append(values, map[string]any{
				"context":          v.Context,
				"contextParameter": v.ContextParameter,
				"value":            v.Value,
			})
		}

		updatedBy := ""
		if rec.UpdatedBy != nil {
			updatedBy = rec.UpdatedBy.Email
		}

		id := accountID + "/" + siteID + "/" + rec.Key
		envVar, err := CreateResource(runtime, "netlify.envVar", map[string]*llx.RawData{
			"__id":           llx.StringData(id),
			"key":            llx.StringData(rec.Key),
			"scopes":         llx.ArrayData(strSliceToAny(rec.Scopes), types.String),
			"isSecret":       llx.BoolData(rec.IsSecret),
			"values":         llx.ArrayData(values, types.Dict),
			"updatedByEmail": llx.StringData(updatedBy),
			"updatedAt":      llx.TimeDataPtr(rec.UpdatedAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, envVar)
	}
	return res, nil
}
