// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/scim"
)

// scimPageSize is the per-request limit used when paginating SCIM endpoints.
// 100 is the conservative ceiling — some Atlassian SCIM deployments cap count
// at 100 server-side.
const scimPageSize = 100

func (a *mqlAtlassianScim) id() (string, error) {
	return "scim", nil
}

func (a *mqlAtlassianScim) users() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*scim.ScimConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow scim access")
	}
	admin := conn.Client()
	directoryID := conn.Directory()
	res := []any{}
	// SCIM 2.0 (RFC 7644 §3.4.2.4) uses 1-based pagination.
	startIndex := 1
	for {
		page, _, err := admin.SCIM.User.Gets(context.Background(), directoryID, nil, startIndex, scimPageSize)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Resources) == 0 {
			break
		}
		for _, scimUser := range page.Resources {
			if scimUser == nil {
				continue
			}
			formatted := ""
			if scimUser.Name != nil {
				formatted = scimUser.Name.Formatted
			}
			mqlUser, err := CreateResource(a.MqlRuntime, "atlassian.scim.user",
				map[string]*llx.RawData{
					"id":           llx.StringData(scimUser.ID),
					"name":         llx.StringData(formatted),
					"displayName":  llx.StringData(scimUser.DisplayName),
					"organization": llx.StringData(scimUser.Organization),
					"title":        llx.StringData(scimUser.Title),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlUser)
		}
		if len(page.Resources) < scimPageSize {
			break
		}
		startIndex += len(page.Resources)
	}
	return res, nil
}

func (a *mqlAtlassianScim) groups() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*scim.ScimConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow scim access")
	}
	admin := conn.Client()
	directoryID := conn.Directory()
	res := []any{}
	// SCIM 2.0 (RFC 7644 §3.4.2.4) uses 1-based pagination.
	startIndex := 1
	for {
		page, _, err := admin.SCIM.Group.Gets(context.Background(), directoryID, "", startIndex, scimPageSize)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Resources) == 0 {
			break
		}
		for _, scimGroup := range page.Resources {
			if scimGroup == nil {
				continue
			}
			mqlGroup, err := CreateResource(a.MqlRuntime, "atlassian.scim.group",
				map[string]*llx.RawData{
					"id":   llx.StringData(scimGroup.ID),
					"name": llx.StringData(scimGroup.DisplayName),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlGroup)
		}
		if len(page.Resources) < scimPageSize {
			break
		}
		startIndex += len(page.Resources)
	}
	return res, nil
}

func (a *mqlAtlassianScimUser) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAtlassianScimGroup) id() (string, error) {
	return a.Id.Data, nil
}
