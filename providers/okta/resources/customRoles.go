// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/providers/okta/resources/sdk"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOkta) customRoles() ([]any, error) {
	runtime := o.MqlRuntime

	conn := runtime.Connection.(*connection.OktaConnection)

	ctx := context.Background()
	apiSupplement := &sdk.ApiExtension{
		Host:  conn.OrganizationID(),
		Token: conn.Token(),
	}

	respList, resp, err := apiSupplement.ListCustomRoles(ctx)
	if err != nil {
		// handle case where no custom roles exist
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(respList) == 0 {
		return nil, nil
	}

	list := []any{}
	for i := range respList {
		r, err := newMqlOktaCustomRole(o.MqlRuntime, respList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func newMqlOktaCustomRole(runtime *plugin.Runtime, entry *sdk.CustomRole) (any, error) {
	return CreateResource(runtime, "okta.customRole", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"label":       llx.StringData(entry.Label),
		"description": llx.StringData(entry.Description),
		"permissions": llx.ArrayData(convert.SliceAnyToInterface(entry.Permissions), types.String),
	})
}

func (o *mqlOktaRole) id() (string, error) {
	return "okta.role/" + o.Id.Data, o.Id.Error
}
