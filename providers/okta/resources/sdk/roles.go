// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"net/http"
)

type ListCustomRolesResponse struct {
	Roles []*CustomRole `json:"roles,omitempty"`
}

type CustomRole struct {
	Id          string      `json:"id,omitempty"`
	Label       string      `json:"label,omitempty"`
	Description string      `json:"description,omitempty"`
	Permissions []string    `json:"permissions,omitempty"`
	Links       interface{} `json:"_links,omitempty"`
}

// ListCustomRoles Gets all customRoles based on the query params
func (m *ApiExtension) ListCustomRoles(ctx context.Context, qp *query.Params) (*ListCustomRolesResponse, *okta.Response, error) {
	url := "/api/v1/iam/roles"
	if qp != nil {
		url += qp.String()
	}
	rq := m.RequestExecutor
	req, err := rq.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	var response *ListCustomRolesResponse
	resp, err := rq.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}
	return response, resp, nil
}
