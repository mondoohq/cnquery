// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"log"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
)

func (a *mqlMicrosoftConditionalAccess) namedLocations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	// Make a request to get named locations
	ctx := context.Background()
	namedLocations, err := graphClient.Identity().ConditionalAccess().NamedLocations().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	// Collect all named location names
	var locationNames []interface{} // Changed to interface{} to match the expected return type
	for _, location := range namedLocations.GetValue() {
		// Use type assertion to check for IP named locations
		if ipLocation, ok := location.(*models.IpNamedLocation); ok {
			displayName := ipLocation.GetDisplayName()
			if displayName != nil {
				locationNames = append(locationNames, *displayName)
			}
		}
	}

	if len(locationNames) == 0 {
		log.Println("No named locations are defined.")
		return nil, nil
	}

	return locationNames, nil
}
