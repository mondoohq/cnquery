// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
)

func (c *mqlNetworkRoutes) list() ([]any, error) {
	return c.List.Data, c.List.Error
}

func (c *mqlNetworkRoutes) defaults() ([]any, error) {
	log.Debug().Msg("os.network.routes> defaults")

	// Get all routes from the list
	allRoutes := c.GetList()
	if allRoutes.Error != nil {
		return nil, allRoutes.Error
	}

	// Filter to only default routes
	var defaultRoutes []any
	for _, routeRes := range allRoutes.Data {
		route, ok := routeRes.(*mqlNetworkRoute)
		if !ok {
			continue
		}

		dest := route.GetDestination()
		if dest.Error != nil {
			continue
		}

		// Check if it's a default route
		destStr := dest.Data
		if destStr == "0.0.0.0/0" || destStr == "::/0" || destStr == "default" ||
			destStr == "0.0.0.0" || destStr == "::" {
			defaultRoutes = append(defaultRoutes, routeRes)
		}
	}

	return defaultRoutes, nil
}

func (c *mqlNetworkRoute) iface() (*mqlNetworkInterface, error) {
	return c.Iface.Data, c.Iface.Error
}
