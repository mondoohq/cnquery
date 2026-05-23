// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

func (c *mqlCloudflareTunnel) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareTunnelConnection) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) tunnels() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	accountID := acc.Data.GetId().Data

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.ListTunnels(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: accountID,
			Level:      cloudflare.AccountRouteLevel,
		}, cloudflare.TunnelListParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			connections := make([]any, 0, len(rec.Connections))
			for j := range rec.Connections {
				tc := rec.Connections[j]

				connRes, err := NewResource(c.MqlRuntime, "cloudflare.tunnel.connection", map[string]*llx.RawData{
					"__id":               llx.StringData(fmt.Sprintf("tunnelConn@%s@%s", rec.ID, tc.ID)),
					"id":                 llx.StringData(tc.ID),
					"coloName":           llx.StringData(tc.ColoName),
					"clientId":           llx.StringData(tc.ClientID),
					"clientVersion":      llx.StringData(tc.ClientVersion),
					"openedAt":           llx.TimeData(parseRFC3339(tc.OpenedAt)),
					"originIp":           llx.StringData(tc.OriginIP),
					"isPendingReconnect": llx.BoolData(tc.IsPendingReconnect),
				})
				if err != nil {
					return nil, err
				}
				connections = append(connections, connRes)
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.tunnel", map[string]*llx.RawData{
				"id":           llx.StringData(rec.ID),
				"name":         llx.StringData(rec.Name),
				"tunnelType":   llx.StringData(rec.TunnelType),
				"status":       llx.StringData(rec.Status),
				"remoteConfig": llx.BoolData(rec.RemoteConfig),
				"createdAt":    llx.TimeDataPtr(rec.CreatedAt),
				"deletedAt":    llx.TimeDataPtr(rec.DeletedAt),
				"connections":  llx.ArrayData(connections, "cloudflare.tunnel.connection"),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}

type mqlCloudflareTunnelRouteInternal struct {
	cacheVirtualNetworkID string
}

func (c *mqlCloudflareTunnelRoute) virtualNetwork() (*mqlCloudflareTunnelVirtualNetwork, error) {
	if c.cacheVirtualNetworkID == "" {
		c.VirtualNetwork.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(c.MqlRuntime, "cloudflare.tunnel.virtualNetwork", map[string]*llx.RawData{
		"id": llx.StringData(c.cacheVirtualNetworkID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlCloudflareTunnelVirtualNetwork), nil
}

func (c *mqlCloudflareTunnelRoute) id() (string, error) {
	if c.Network.Error != nil {
		return "", c.Network.Error
	}
	if c.TunnelId.Error != nil {
		return "", c.TunnelId.Error
	}
	return tunnelRouteID(c.Network.Data, c.TunnelId.Data, c.cacheVirtualNetworkID), nil
}

func tunnelRouteID(network, tunnelID, vnetID string) string {
	return fmt.Sprintf("tunnelRoute@%s@%s@%s", network, tunnelID, vnetID)
}

func (c *mqlCloudflareZone) tunnelRoutes() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	accountID := acc.Data.GetId().Data

	const perPage = 50
	params := cloudflare.TunnelRoutesListParams{
		PaginationOptions: cloudflare.PaginationOptions{PerPage: perPage, Page: 1},
	}

	var result []any
	for {
		records, err := conn.Cf.ListTunnelRoutes(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: accountID,
			Level:      cloudflare.AccountRouteLevel,
		}, params)
		if err != nil {
			return nil, err
		}

		for i := range records {
			rec := records[i]

			res, err := CreateResource(c.MqlRuntime, "cloudflare.tunnel.route", map[string]*llx.RawData{
				"__id":       llx.StringData(tunnelRouteID(rec.Network, rec.TunnelID, rec.VirtualNetworkID)),
				"network":    llx.StringData(rec.Network),
				"tunnelId":   llx.StringData(rec.TunnelID),
				"tunnelName": llx.StringData(rec.TunnelName),
				"comment":    llx.StringData(rec.Comment),
				"createdAt":  llx.TimeDataPtr(rec.CreatedAt),
				"deletedAt":  llx.TimeDataPtr(rec.DeletedAt),
			})
			if err != nil {
				return nil, err
			}

			res.(*mqlCloudflareTunnelRoute).cacheVirtualNetworkID = rec.VirtualNetworkID
			result = append(result, res)
		}

		if len(records) < perPage {
			break
		}
		params.PaginationOptions.Page++
	}

	return result, nil
}

func (c *mqlCloudflareTunnelVirtualNetwork) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) tunnelVirtualNetworks() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	accountID := acc.Data.GetId().Data

	const perPage = 50
	params := cloudflare.TunnelVirtualNetworksListParams{
		PaginationOptions: cloudflare.PaginationOptions{PerPage: perPage, Page: 1},
	}

	var result []any
	for {
		records, err := conn.Cf.ListTunnelVirtualNetworks(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: accountID,
			Level:      cloudflare.AccountRouteLevel,
		}, params)
		if err != nil {
			return nil, err
		}

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.tunnel.virtualNetwork", map[string]*llx.RawData{
				"id":               llx.StringData(rec.ID),
				"name":             llx.StringData(rec.Name),
				"comment":          llx.StringData(rec.Comment),
				"isDefaultNetwork": llx.BoolData(rec.IsDefaultNetwork),
				"createdAt":        llx.TimeDataPtr(rec.CreatedAt),
				"deletedAt":        llx.TimeDataPtr(rec.DeletedAt),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if len(records) < perPage {
			break
		}
		params.PaginationOptions.Page++
	}

	return result, nil
}
