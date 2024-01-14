// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
	"go.mondoo.com/cnquery/v10/providers/okta/resources/sdk"
	"go.mondoo.com/cnquery/v10/types"
	"net/http"
	"strings"
)

func (o *mqlOkta) customRoles() ([]interface{}, error) {
	runtime := o.MqlRuntime

	conn := runtime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	apiSupplement := &sdk.ApiExtension{
		RequestExecutor: client.CloneRequestExecutor(),
	}

	respList, resp, err := apiSupplement.ListCustomRoles(
		ctx,
		nil,
	)

	// handle case where no policy exists
	if err != nil && resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	// handle special case where the policy type does not exist
	if err != nil && resp.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(err.Error()), "invalid policy type") {
		return nil, nil
	}

	if len(respList.Roles) == 0 {
		return nil, nil
	}

	list := []interface{}{}
	appendEntry := func(datalist []*sdk.CustomRole) error {
		for i := range datalist {
			r, err := newMqlOktaCustomRole(o.MqlRuntime, datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	err = appendEntry(respList.Roles)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var roles []*sdk.CustomRole
		resp, err = resp.Next(ctx, &roles)
		if err != nil {
			return nil, err
		}
		err = appendEntry(roles)
		if err != nil {
			return nil, err
		}
	}

	return list, nil

}

func newMqlOktaCustomRole(runtime *plugin.Runtime, entry *sdk.CustomRole) (interface{}, error) {
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
