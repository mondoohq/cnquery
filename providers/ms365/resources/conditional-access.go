// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

func (a *mqlMicrosoftConditionalAccess) namedLocations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	namedLocations, err := graphClient.Identity().ConditionalAccess().NamedLocations().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var locationDetails []interface{}
	for _, location := range namedLocations.GetValue() {
		if ipLocation, ok := location.(*models.IpNamedLocation); ok {
			displayName := ipLocation.GetDisplayName()
			isTrusted := ipLocation.GetIsTrusted()

			if displayName != nil {
				trusted := false
				if isTrusted != nil {
					trusted = *isTrusted
				}

				locationInfo, err := CreateResource(a.MqlRuntime, "microsoft.conditionalAccess.ipNamedLocation",
					map[string]*llx.RawData{
						"name":    llx.StringDataPtr(displayName),
						"trusted": llx.BoolData(trusted),
					})
				if err != nil {
					return nil, err
				}
				locationDetails = append(locationDetails, locationInfo)
			}
		}
	}

	if len(locationDetails) == 0 {
		return nil, nil
	}

	return locationDetails, nil
}
