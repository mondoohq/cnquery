// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/groupsettings"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

func (a *mqlMicrosoft) settings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	settings, err := graphClient.GroupSettings().Get(ctx, &groupsettings.GroupSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}
	return convert.JsonToDictSlice(newSettings(settings.GetValue()))
}
