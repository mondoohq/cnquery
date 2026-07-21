// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"fmt"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/zero_trust"
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

	var result []any
	iter := conn.Cf.ZeroTrust.Tunnels.Cloudflared.ListAutoPaging(context.TODO(), zero_trust.TunnelCloudflaredListParams{
		AccountID: cloudflare.F(accountID),
	})
	for iter.Next() {
		rec := iter.Current()

		// The inline connections field on the list response is deprecated in
		// cloudflare-go v6 in favor of the dedicated per-tunnel connections
		// endpoint, so fetch connection details from there instead.
		connections, err := c.tunnelConnections(conn, accountID, rec.ID, string(rec.TunType))
		if err != nil {
			return nil, err
		}

		res, err := NewResource(c.MqlRuntime, "cloudflare.tunnel", map[string]*llx.RawData{
			"id":         llx.StringData(rec.ID),
			"name":       llx.StringData(rec.Name),
			"tunnelType": llx.StringData(string(rec.TunType)),
			"status":     llx.StringData(string(rec.Status)),
			// v6 deprecates the remote_config bool in favor of config_src;
			// "cloudflare" means the tunnel is managed remotely from the
			// Zero Trust dashboard, which is what remote_config==true meant.
			"remoteConfig": llx.BoolData(string(rec.ConfigSrc) == "cloudflare"),
			"createdAt":    timeOrNil(rec.CreatedAt),
			"deletedAt":    timeOrNil(rec.DeletedAt),
			"connections":  llx.ArrayData(connections, "cloudflare.tunnel.connection"),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
	}

	return result, nil
}

// tunnelConnections fetches the active connections for a single tunnel from the
// dedicated connections endpoint. This endpoint only serves cloudflared
// ("cfd_tunnel") tunnels; other tunnel types (e.g. warp_connector) have no
// connections here, so we return an empty list for them.
func (c *mqlCloudflareZone) tunnelConnections(conn *connection.CloudflareConnection, accountID, tunnelID, tunType string) ([]any, error) {
	connections := []any{}
	if tunType != "cfd_tunnel" {
		return connections, nil
	}

	iter := conn.Cf.ZeroTrust.Tunnels.Cloudflared.Connections.GetAutoPaging(context.TODO(), tunnelID, zero_trust.TunnelCloudflaredConnectionGetParams{
		AccountID: cloudflare.F(accountID),
	})
	for iter.Next() {
		client := iter.Current()
		for j := range client.Conns {
			tc := client.Conns[j]

			connRes, err := NewResource(c.MqlRuntime, "cloudflare.tunnel.connection", map[string]*llx.RawData{
				"__id":               llx.StringData(fmt.Sprintf("tunnelConn@%s@%s", tunnelID, tc.ID)),
				"id":                 llx.StringData(tc.ID),
				"coloName":           llx.StringData(tc.ColoName),
				"clientId":           llx.StringData(tc.ClientID),
				"clientVersion":      llx.StringData(tc.ClientVersion),
				"openedAt":           timeOrNil(tc.OpenedAt),
				"originIp":           llx.StringData(tc.OriginIP),
				"isPendingReconnect": llx.BoolData(tc.IsPendingReconnect),
			})
			if err != nil {
				return nil, err
			}
			connections = append(connections, connRes)
		}
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
	}

	return connections, nil
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

// tunnelRoute mirrors a Cloudflare Tunnel (teamnet) route. The endpoint is
// page-numbered without a total_pages count, so we decode it via the client's
// generic Get and stop on the first short page.
type tunnelRouteRecord struct {
	Network          string    `json:"network"`
	TunnelID         string    `json:"tunnel_id"`
	TunnelName       string    `json:"tunnel_name"`
	Comment          string    `json:"comment"`
	VirtualNetworkID string    `json:"virtual_network_id"`
	CreatedAt        time.Time `json:"created_at"`
	DeletedAt        time.Time `json:"deleted_at"`
}

func (c *mqlCloudflareZone) tunnelRoutes() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)
	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	accountID := acc.Data.GetId().Data

	const perPage = 50
	var result []any
	page := 1
	for {
		var env struct {
			Result []tunnelRouteRecord `json:"result"`
		}
		uri := fmt.Sprintf("accounts/%s/teamnet/routes?per_page=%d&page=%d", accountID, perPage, page)
		if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
			return degradedList(err)
		}

		for i := range env.Result {
			rec := env.Result[i]

			res, err := CreateResource(c.MqlRuntime, "cloudflare.tunnel.route", map[string]*llx.RawData{
				"__id":       llx.StringData(tunnelRouteID(rec.Network, rec.TunnelID, rec.VirtualNetworkID)),
				"network":    llx.StringData(rec.Network),
				"tunnelId":   llx.StringData(rec.TunnelID),
				"tunnelName": llx.StringData(rec.TunnelName),
				"comment":    llx.StringData(rec.Comment),
				"createdAt":  timeOrNil(rec.CreatedAt),
				"deletedAt":  timeOrNil(rec.DeletedAt),
			})
			if err != nil {
				return nil, err
			}

			res.(*mqlCloudflareTunnelRoute).cacheVirtualNetworkID = rec.VirtualNetworkID
			result = append(result, res)
		}

		if len(env.Result) < perPage {
			break
		}
		page++
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

	var result []any
	iter := conn.Cf.ZeroTrust.Networks.VirtualNetworks.ListAutoPaging(context.TODO(), zero_trust.NetworkVirtualNetworkListParams{
		AccountID: cloudflare.F(accountID),
	})
	for iter.Next() {
		rec := iter.Current()

		res, err := NewResource(c.MqlRuntime, "cloudflare.tunnel.virtualNetwork", map[string]*llx.RawData{
			"id":               llx.StringData(rec.ID),
			"name":             llx.StringData(rec.Name),
			"comment":          llx.StringData(rec.Comment),
			"isDefaultNetwork": llx.BoolData(rec.IsDefaultNetwork),
			"createdAt":        timeOrNil(rec.CreatedAt),
			"deletedAt":        timeOrNil(rec.DeletedAt),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}
	if err := iter.Err(); err != nil {
		return degradedList(err)
	}

	return result, nil
}
