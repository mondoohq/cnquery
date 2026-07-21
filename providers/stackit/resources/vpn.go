// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlStackitVpnGatewayConnectionInternal struct {
	// cacheTunnel1/cacheTunnel2 hold the connection's two tunnel
	// configurations, captured when the connection is built so tunnel1() and
	// tunnel2() can expose them without another API call. cacheIdBase is the
	// connection's own cache key, used to key the tunnel sub-resources.
	cacheTunnel1 *vpn.TunnelConfiguration
	cacheTunnel2 *vpn.TunnelConfiguration
	cacheIdBase  string
}

func (r *mqlStackit) vpn() (*mqlStackitVpn, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.vpn")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitVpn), nil
}

// ------------------------- gateways -------------------------

func (r *mqlStackitVpn) gateways() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Vpn()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListGateways(bgctx(), c.ProjectID(), c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetGatewaysOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildVpnGateway(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildVpnGateway(runtime *plugin.Runtime, g *vpn.GatewayResponse) (plugin.Resource, error) {
	var (
		bgpLocalAsn int64
		bgpRoutes   []string
	)
	if bgp, ok := g.GetBgpOk(); ok && bgp != nil {
		bgpLocalAsn = bgp.GetLocalAsn()
		bgpRoutes = bgp.GetOverrideAdvertisedRoutes()
	}
	az := g.GetAvailabilityZones()
	args := map[string]*llx.RawData{
		"id":                          llx.StringData(g.GetId()),
		"name":                        llx.StringData(g.GetDisplayName()),
		"status":                      llx.StringData(string(g.GetState())),
		"routingType":                 llx.StringData(string(g.GetRoutingType())),
		"planId":                      llx.StringData(g.GetPlanId()),
		"tunnel1AvailabilityZone":     llx.StringData(az.GetTunnel1()),
		"tunnel2AvailabilityZone":     llx.StringData(az.GetTunnel2()),
		"bgpLocalAsn":                 llx.IntData(bgpLocalAsn),
		"bgpOverrideAdvertisedRoutes": strSliceData(bgpRoutes),
		"labels":                      labelData(g.GetLabels()),
	}
	return CreateResource(runtime, "stackit.vpn.gateway", args)
}

func (r *mqlStackitVpnGateway) id() (string, error) {
	return "stackit.vpn.gateway/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Id.Data, nil
}

func initStackitVpnGateway(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Vpn()
	if err != nil {
		return nil, nil, err
	}
	g, err := client.DefaultAPI.GetGateway(bgctx(), c.ProjectID(), c.Region(), id).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := buildVpnGateway(runtime, g)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- connections -------------------------

func (r *mqlStackitVpnGateway) connections() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Vpn()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListGatewayConnections(bgctx(), c.ProjectID(), c.Region(), r.Id.Data).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetConnectionsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		connResp := items[i]
		idBase := "stackit.vpn.gateway.connection/" + c.ProjectID() + "/" + r.Id.Data + "/" + connResp.GetId()
		res, err := CreateResource(r.MqlRuntime, "stackit.vpn.gateway.connection", map[string]*llx.RawData{
			"__id":          llx.StringData(idBase),
			"id":            llx.StringData(connResp.GetId()),
			"name":          llx.StringData(connResp.GetDisplayName()),
			"enabled":       llx.BoolData(connResp.GetEnabled()),
			"localSubnets":  strSliceData(connResp.GetLocalSubnets()),
			"remoteSubnets": strSliceData(connResp.GetRemoteSubnets()),
			"staticRoutes":  strSliceData(connResp.GetStaticRoutes()),
			"labels":        labelData(connResp.GetLabels()),
		})
		if err != nil {
			return nil, err
		}
		mqlConn := res.(*mqlStackitVpnGatewayConnection)
		t1 := connResp.GetTunnel1()
		t2 := connResp.GetTunnel2()
		mqlConn.cacheTunnel1 = &t1
		mqlConn.cacheTunnel2 = &t2
		mqlConn.cacheIdBase = idBase
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitVpnGatewayConnection) tunnel1() (*mqlStackitVpnTunnel, error) {
	if r.cacheTunnel1 == nil {
		return markNull[mqlStackitVpnTunnel](&r.Tunnel1)
	}
	return r.buildTunnel(r.cacheTunnel1, "tunnel1")
}

func (r *mqlStackitVpnGatewayConnection) tunnel2() (*mqlStackitVpnTunnel, error) {
	if r.cacheTunnel2 == nil {
		return markNull[mqlStackitVpnTunnel](&r.Tunnel2)
	}
	return r.buildTunnel(r.cacheTunnel2, "tunnel2")
}

func (r *mqlStackitVpnGatewayConnection) buildTunnel(t *vpn.TunnelConfiguration, slot string) (*mqlStackitVpnTunnel, error) {
	var (
		bgpRemoteAsn         int64
		peeringLocalAddress  string
		peeringRemoteAddress string
	)
	if bgp, ok := t.GetBgpOk(); ok && bgp != nil {
		bgpRemoteAsn = bgp.GetRemoteAsn()
	}
	if peering, ok := t.GetPeeringOk(); ok && peering != nil {
		peeringLocalAddress = peering.GetLocalAddress()
		peeringRemoteAddress = peering.GetRemoteAddress()
	}
	p1 := t.GetPhase1()
	p2 := t.GetPhase2()
	res, err := CreateResource(r.MqlRuntime, "stackit.vpn.tunnel", map[string]*llx.RawData{
		"__id":                       llx.StringData(r.cacheIdBase + "/" + slot),
		"remoteAddress":              llx.StringData(t.GetRemoteAddress()),
		"bgpRemoteAsn":               llx.IntData(bgpRemoteAsn),
		"peeringLocalAddress":        llx.StringData(peeringLocalAddress),
		"peeringRemoteAddress":       llx.StringData(peeringRemoteAddress),
		"phase1DhGroups":             strSliceData(enumSliceToStr(p1.GetDhGroups())),
		"phase1EncryptionAlgorithms": strSliceData(enumSliceToStr(p1.GetEncryptionAlgorithms())),
		"phase1IntegrityAlgorithms":  strSliceData(enumSliceToStr(p1.GetIntegrityAlgorithms())),
		"phase1RekeyTime":            llx.IntData(int64(p1.GetRekeyTime())),
		"phase2DhGroups":             strSliceData(enumSliceToStr(p2.GetDhGroups())),
		"phase2EncryptionAlgorithms": strSliceData(enumSliceToStr(p2.GetEncryptionAlgorithms())),
		"phase2IntegrityAlgorithms":  strSliceData(enumSliceToStr(p2.GetIntegrityAlgorithms())),
		"phase2RekeyTime":            llx.IntData(int64(p2.GetRekeyTime())),
		"dpdAction":                  llx.StringData(string(p2.GetDpdAction())),
		"startAction":                llx.StringData(string(p2.GetStartAction())),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitVpnTunnel), nil
}

// enumSliceToStr converts a slice of string-based SDK enum values into a
// plain []string.
func enumSliceToStr[T ~string](in []T) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}
