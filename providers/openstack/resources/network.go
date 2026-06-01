// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	neutronquotas "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/quotas"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// ---- openstack.network ----

type mqlOpenstackNetworkInternal struct {
	cacheSubnetIDs []string
}

// networkExt is the union of the base network with provider:*, router:external,
// and mtu extensions so a single list call returns all the fields we expose.
type networkExt struct {
	networks.Network
	provider.NetworkProviderExt
	external.NetworkExternalExt
	mtu.NetworkMTUExt
}

// portExt is the base port plus the port-security extension; it lets one list
// call surface portSecurityEnabled. Clouds without the extension simply leave
// the field at false.
type portExt struct {
	ports.Port
	portsecurity.PortSecurityExt
}

func (r *mqlOpenstackNetwork) id() (string, error) {
	return "openstack.network/" + r.Id.Data, nil
}

func initOpenstackNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetNetworks()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		n := raw.(*mqlOpenstackNetwork)
		if n.Id.Data == id {
			return args, n, nil
		}
	}
	initSyntheticID("openstack.network", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) networks() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := networks.List(client, networks.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}

	var items []networkExt
	if err := networks.ExtractNetworksInto(pages, &items); err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		n := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.network", map[string]*llx.RawData{
			"__id":                    llx.StringData("openstack.network/" + n.ID),
			"id":                      llx.StringData(n.ID),
			"name":                    llx.StringData(n.Name),
			"status":                  llx.StringData(n.Status),
			"adminStateUp":            llx.BoolData(n.AdminStateUp),
			"shared":                  llx.BoolData(n.Shared),
			"external":                llx.BoolData(n.External),
			"mtu":                     llx.IntData(int64(n.MTU)),
			"projectId":               llx.StringData(n.ProjectID),
			"description":             llx.StringData(n.Description),
			"tags":                    stringSliceData(n.Tags),
			"providerNetworkType":     llx.StringData(n.NetworkType),
			"providerPhysicalNetwork": llx.StringData(n.PhysicalNetwork),
			"providerSegmentationId":  llx.IntData(parseSegmentationID(n.SegmentationID)),
			"createdAt":               llx.TimeDataPtr(timePtr(n.CreatedAt)),
			"updatedAt":               llx.TimeDataPtr(timePtr(n.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlNetwork := res.(*mqlOpenstackNetwork)
		mqlNetwork.cacheSubnetIDs = n.Subnets
		out = append(out, mqlNetwork)
	}
	return out, nil
}

func (r *mqlOpenstackNetwork) subnets() ([]any, error) {
	if len(r.cacheSubnetIDs) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(r.cacheSubnetIDs))
	for _, id := range r.cacheSubnetIDs {
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.subnet", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// parseSegmentationID coerces the provider:segmentation_id field into int64.
// Neutron returns it as a string in some clouds and a number in others, and
// gophercloud's NetworkProviderExt declares it as `string`.
func parseSegmentationID(s string) int64 {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int64(ch-'0')
	}
	return n
}

// ---- openstack.subnet ----

type mqlOpenstackSubnetInternal struct {
	cacheNetworkID    string
	cacheSubnetPoolID string
}

func (r *mqlOpenstackSubnet) id() (string, error) {
	return "openstack.subnet/" + r.Id.Data, nil
}

func initOpenstackSubnet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetSubnets()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		s := raw.(*mqlOpenstackSubnet)
		if s.Id.Data == id {
			return args, s, nil
		}
	}
	initSyntheticID("openstack.subnet", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) subnets() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := subnets.List(client, subnets.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := subnets.ExtractSubnets(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		s := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.subnet", map[string]*llx.RawData{
			"__id":              llx.StringData("openstack.subnet/" + s.ID),
			"id":                llx.StringData(s.ID),
			"name":              llx.StringData(s.Name),
			"cidr":              llx.StringData(s.CIDR),
			"ipVersion":         llx.IntData(int64(s.IPVersion)),
			"gatewayIp":         llx.StringData(s.GatewayIP),
			"enableDhcp":        llx.BoolData(s.EnableDHCP),
			"dnsPublishFixedIp": llx.BoolData(s.DNSPublishFixedIP),
			"ipv6AddressMode":   llx.StringData(s.IPv6AddressMode),
			"ipv6RaMode":        llx.StringData(s.IPv6RAMode),
			"dnsNameservers":    stringSliceData(s.DNSNameservers),
			"allocationPools":   dictSliceData(allocationPoolsToDict(s.AllocationPools)),
			"hostRoutes":        dictSliceData(hostRoutesToDict(s.HostRoutes)),
			"projectId":         llx.StringData(s.ProjectID),
			"description":       llx.StringData(s.Description),
			"tags":              stringSliceData(s.Tags),
			"createdAt":         llx.TimeDataPtr(timePtr(s.CreatedAt)),
			"updatedAt":         llx.TimeDataPtr(timePtr(s.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlSubnet := res.(*mqlOpenstackSubnet)
		mqlSubnet.cacheNetworkID = s.NetworkID
		mqlSubnet.cacheSubnetPoolID = s.SubnetPoolID
		out = append(out, mqlSubnet)
	}
	return out, nil
}

func (r *mqlOpenstackSubnet) network() (*mqlOpenstackNetwork, error) {
	if r.cacheNetworkID == "" {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.network", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheNetworkID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackNetwork), nil
}

func (r *mqlOpenstackSubnet) subnetPool() (*mqlOpenstackSubnetPool, error) {
	if r.cacheSubnetPoolID == "" {
		r.SubnetPool.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.subnetPool", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheSubnetPoolID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackSubnetPool), nil
}

func allocationPoolsToDict(pools []subnets.AllocationPool) []any {
	out := make([]any, 0, len(pools))
	for _, p := range pools {
		out = append(out, map[string]any{
			"start": p.Start,
			"end":   p.End,
		})
	}
	return out
}

func hostRoutesToDict(routes []subnets.HostRoute) []any {
	out := make([]any, 0, len(routes))
	for _, r := range routes {
		out = append(out, map[string]any{
			"destination": r.DestinationCIDR,
			"nexthop":     r.NextHop,
		})
	}
	return out
}

// ---- openstack.router ----

type mqlOpenstackRouterInternal struct {
	cacheExternalNetworkID string
}

func (r *mqlOpenstackRouter) id() (string, error) {
	return "openstack.router/" + r.Id.Data, nil
}

func initOpenstackRouter(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetRouters()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		r := raw.(*mqlOpenstackRouter)
		if r.Id.Data == id {
			return args, r, nil
		}
	}
	initSyntheticID("openstack.router", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) routers() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := routers.List(client, routers.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := routers.ExtractRouters(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		rt := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.router", map[string]*llx.RawData{
			"__id":                llx.StringData("openstack.router/" + rt.ID),
			"id":                  llx.StringData(rt.ID),
			"name":                llx.StringData(rt.Name),
			"status":              llx.StringData(rt.Status),
			"adminStateUp":        llx.BoolData(rt.AdminStateUp),
			"distributed":         llx.BoolData(rt.Distributed),
			"projectId":           llx.StringData(rt.ProjectID),
			"description":         llx.StringData(rt.Description),
			"tags":                stringSliceData(rt.Tags),
			"externalGatewayInfo": llx.DictData(gatewayInfoToDict(rt.GatewayInfo)),
			"routes":              dictSliceData(routesToDict(rt.Routes)),
			"createdAt":           llx.TimeDataPtr(timePtr(rt.CreatedAt)),
			"updatedAt":           llx.TimeDataPtr(timePtr(rt.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlRouter := res.(*mqlOpenstackRouter)
		mqlRouter.cacheExternalNetworkID = rt.GatewayInfo.NetworkID
		out = append(out, mqlRouter)
	}
	return out, nil
}

func (r *mqlOpenstackRouter) externalNetwork() (*mqlOpenstackNetwork, error) {
	if r.cacheExternalNetworkID == "" {
		r.ExternalNetwork.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.network", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheExternalNetworkID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackNetwork), nil
}

func gatewayInfoToDict(g routers.GatewayInfo) map[string]any {
	if g.NetworkID == "" && len(g.ExternalFixedIPs) == 0 {
		return nil
	}
	out := map[string]any{
		"network_id": g.NetworkID,
	}
	if g.EnableSNAT != nil {
		out["enable_snat"] = *g.EnableSNAT
	}
	if len(g.ExternalFixedIPs) > 0 {
		fixed := make([]any, 0, len(g.ExternalFixedIPs))
		for _, fip := range g.ExternalFixedIPs {
			fixed = append(fixed, map[string]any{
				"ip_address": fip.IPAddress,
				"subnet_id":  fip.SubnetID,
			})
		}
		out["external_fixed_ips"] = fixed
	}
	return out
}

func routesToDict(in []routers.Route) []any {
	out := make([]any, 0, len(in))
	for _, r := range in {
		out = append(out, map[string]any{
			"destination": r.DestinationCIDR,
			"nexthop":     r.NextHop,
		})
	}
	return out
}

// ---- openstack.port ----

type mqlOpenstackPortInternal struct {
	cacheNetworkID        string
	cacheSecurityGroupIDs []string
}

func (r *mqlOpenstackPort) id() (string, error) {
	return "openstack.port/" + r.Id.Data, nil
}

func initOpenstackPort(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetPorts()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		p := raw.(*mqlOpenstackPort)
		if p.Id.Data == id {
			return args, p, nil
		}
	}
	initSyntheticID("openstack.port", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) ports() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := ports.List(client, ports.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}

	var items []portExt
	if err := ports.ExtractPortsInto(pages, &items); err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		p := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.port", map[string]*llx.RawData{
			"__id":                  llx.StringData("openstack.port/" + p.ID),
			"id":                    llx.StringData(p.ID),
			"name":                  llx.StringData(p.Name),
			"status":                llx.StringData(p.Status),
			"adminStateUp":          llx.BoolData(p.AdminStateUp),
			"macAddress":            llx.StringData(p.MACAddress),
			"deviceId":              llx.StringData(p.DeviceID),
			"deviceOwner":           llx.StringData(p.DeviceOwner),
			"fixedIps":              dictSliceData(fixedIPsToDict(p.FixedIPs)),
			"portSecurityEnabled":   llx.BoolData(p.PortSecurityEnabled),
			"propagateUplinkStatus": llx.BoolData(p.PropagateUplinkStatus),
			"projectId":             llx.StringData(p.ProjectID),
			"description":           llx.StringData(p.Description),
			"tags":                  stringSliceData(p.Tags),
			"createdAt":             llx.TimeDataPtr(timePtr(p.CreatedAt)),
			"updatedAt":             llx.TimeDataPtr(timePtr(p.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlPort := res.(*mqlOpenstackPort)
		mqlPort.cacheNetworkID = p.NetworkID
		mqlPort.cacheSecurityGroupIDs = p.SecurityGroups
		out = append(out, mqlPort)
	}
	return out, nil
}

func (r *mqlOpenstackPort) network() (*mqlOpenstackNetwork, error) {
	if r.cacheNetworkID == "" {
		r.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.network", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheNetworkID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackNetwork), nil
}

func (r *mqlOpenstackPort) securityGroups() ([]any, error) {
	if len(r.cacheSecurityGroupIDs) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(r.cacheSecurityGroupIDs))
	for _, id := range r.cacheSecurityGroupIDs {
		if id == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "openstack.securityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func fixedIPsToDict(in []ports.IP) []any {
	out := make([]any, 0, len(in))
	for _, ip := range in {
		out = append(out, map[string]any{
			"subnet_id":  ip.SubnetID,
			"ip_address": ip.IPAddress,
		})
	}
	return out
}

// ---- openstack.floatingIp ----

type mqlOpenstackFloatingIpInternal struct {
	cacheFloatingNetworkID string
	cacheRouterID          string
	cachePortID            string
}

func (r *mqlOpenstackFloatingIp) id() (string, error) {
	return "openstack.floatingIp/" + r.Id.Data, nil
}

func initOpenstackFloatingIp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetFloatingIps()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		f := raw.(*mqlOpenstackFloatingIp)
		if f.Id.Data == id {
			return args, f, nil
		}
	}
	initSyntheticID("openstack.floatingIp", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) floatingIps() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := floatingips.List(client, floatingips.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := floatingips.ExtractFloatingIPs(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		f := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.floatingIp", map[string]*llx.RawData{
			"__id":              llx.StringData("openstack.floatingIp/" + f.ID),
			"id":                llx.StringData(f.ID),
			"status":            llx.StringData(f.Status),
			"floatingIpAddress": llx.StringData(f.FloatingIP),
			"fixedIpAddress":    llx.StringData(f.FixedIP),
			"projectId":         llx.StringData(f.ProjectID),
			"description":       llx.StringData(f.Description),
			"tags":              stringSliceData(f.Tags),
			"createdAt":         llx.TimeDataPtr(timePtr(f.CreatedAt)),
			"updatedAt":         llx.TimeDataPtr(timePtr(f.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlFip := res.(*mqlOpenstackFloatingIp)
		mqlFip.cacheFloatingNetworkID = f.FloatingNetworkID
		mqlFip.cacheRouterID = f.RouterID
		mqlFip.cachePortID = f.PortID
		out = append(out, mqlFip)
	}
	return out, nil
}

func (r *mqlOpenstackFloatingIp) floatingNetwork() (*mqlOpenstackNetwork, error) {
	if r.cacheFloatingNetworkID == "" {
		r.FloatingNetwork.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.network", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheFloatingNetworkID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackNetwork), nil
}

func (r *mqlOpenstackFloatingIp) port() (*mqlOpenstackPort, error) {
	if r.cachePortID == "" {
		r.Port.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.port", map[string]*llx.RawData{
		"id": llx.StringData(r.cachePortID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackPort), nil
}

func (r *mqlOpenstackFloatingIp) router() (*mqlOpenstackRouter, error) {
	if r.cacheRouterID == "" {
		r.Router.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.router", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheRouterID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackRouter), nil
}

// ---- openstack.securityGroup ----

func (r *mqlOpenstackSecurityGroup) id() (string, error) {
	return "openstack.securityGroup/" + r.Id.Data, nil
}

func initOpenstackSecurityGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetSecurityGroups()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		sg := raw.(*mqlOpenstackSecurityGroup)
		if sg.Id.Data == id {
			return args, sg, nil
		}
	}

	// Cross-project rule references may point at a security group outside
	// the locally listed set. Fall back to a direct Neutron Get so the
	// caller observes real fields instead of a synthetic stub.
	c := conn(runtime)
	client, err := c.NetworkClient()
	if err != nil {
		initSyntheticID("openstack.securityGroup", "id", args)
		return args, nil, nil
	}
	sg, err := groups.Get(ctx(), client, id).Extract()
	if err != nil {
		if translateGetError(err) == nil {
			initSyntheticID("openstack.securityGroup", "id", args)
			return args, nil, nil
		}
		return nil, nil, err
	}
	args["__id"] = llx.StringData("openstack.securityGroup/" + sg.ID)
	args["id"] = llx.StringData(sg.ID)
	args["name"] = llx.StringData(sg.Name)
	args["description"] = llx.StringData(sg.Description)
	args["stateful"] = llx.BoolData(sg.Stateful)
	args["projectId"] = llx.StringData(sg.ProjectID)
	args["tags"] = stringSliceData(sg.Tags)
	args["createdAt"] = llx.TimeDataPtr(timePtr(sg.CreatedAt))
	args["updatedAt"] = llx.TimeDataPtr(timePtr(sg.UpdatedAt))
	ruleResources, err := buildSecurityGroupRules(runtime, sg)
	if err != nil {
		return nil, nil, err
	}
	args["rules"] = llx.ArrayData(ruleResources, types.Resource("openstack.securityGroup.rule"))
	return args, nil, nil
}

func (o *mqlOpenstack) securityGroups() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		return nil, err
	}
	pages, err := groups.List(client, groups.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := groups.ExtractGroups(pages)
	if err != nil {
		return nil, err
	}

	// Prime the per-connection name->ID cache from this list call so that
	// Nova security-group-by-name lookups (lookupSecurityGroupIDByName) reuse
	// it instead of issuing a second Neutron list. This accessor is the sole
	// source of the cache: lookupSecurityGroupIDByName routes through here via
	// GetSecurityGroups rather than listing groups itself.
	c.SGNameCacheLock.Lock()
	if c.SGNameCache == nil {
		cache := make(map[string]string, len(items))
		for _, sg := range items {
			cache[sg.Name] = sg.ID
		}
		c.SGNameCache = cache
	}
	c.SGNameCacheLock.Unlock()

	out := make([]any, 0, len(items))
	for i := range items {
		sg := &items[i]
		ruleResources, err := buildSecurityGroupRules(o.MqlRuntime, sg)
		if err != nil {
			return nil, err
		}
		res, err := CreateResource(o.MqlRuntime, "openstack.securityGroup", map[string]*llx.RawData{
			"__id":        llx.StringData("openstack.securityGroup/" + sg.ID),
			"id":          llx.StringData(sg.ID),
			"name":        llx.StringData(sg.Name),
			"description": llx.StringData(sg.Description),
			"stateful":    llx.BoolData(sg.Stateful),
			"projectId":   llx.StringData(sg.ProjectID),
			"tags":        stringSliceData(sg.Tags),
			"createdAt":   llx.TimeDataPtr(timePtr(sg.CreatedAt)),
			"updatedAt":   llx.TimeDataPtr(timePtr(sg.UpdatedAt)),
			"rules":       llx.ArrayData(ruleResources, types.Resource("openstack.securityGroup.rule")),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ---- openstack.securityGroup.rule ----

type mqlOpenstackSecurityGroupRuleInternal struct {
	cacheSecurityGroupID string
	cacheRemoteGroupID   string
}

func (r *mqlOpenstackSecurityGroupRule) id() (string, error) {
	return "openstack.securityGroup.rule/" + r.Id.Data, nil
}

func buildSecurityGroupRules(runtime *plugin.Runtime, sg *groups.SecGroup) ([]any, error) {
	out := make([]any, 0, len(sg.Rules))
	for i := range sg.Rules {
		rule := &sg.Rules[i]
		res, err := CreateResource(runtime, "openstack.securityGroup.rule", map[string]*llx.RawData{
			"__id":           llx.StringData("openstack.securityGroup.rule/" + rule.ID),
			"id":             llx.StringData(rule.ID),
			"direction":      llx.StringData(rule.Direction),
			"ethertype":      llx.StringData(rule.EtherType),
			"protocol":       llx.StringData(rule.Protocol),
			"portRangeMin":   llx.IntData(int64(rule.PortRangeMin)),
			"portRangeMax":   llx.IntData(int64(rule.PortRangeMax)),
			"remoteIpPrefix": llx.StringData(rule.RemoteIPPrefix),
			"projectId":      llx.StringData(rule.ProjectID),
			"description":    llx.StringData(rule.Description),
			"createdAt":      llx.TimeDataPtr(timePtr(rule.CreatedAt)),
			"updatedAt":      llx.TimeDataPtr(timePtr(rule.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := res.(*mqlOpenstackSecurityGroupRule)
		mqlRule.cacheSecurityGroupID = rule.SecGroupID
		mqlRule.cacheRemoteGroupID = rule.RemoteGroupID
		out = append(out, mqlRule)
	}
	return out, nil
}

func (r *mqlOpenstackSecurityGroupRule) securityGroup() (*mqlOpenstackSecurityGroup, error) {
	if r.cacheSecurityGroupID == "" {
		r.SecurityGroup.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.securityGroup", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheSecurityGroupID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackSecurityGroup), nil
}

func (r *mqlOpenstackSecurityGroupRule) remoteGroup() (*mqlOpenstackSecurityGroup, error) {
	if r.cacheRemoteGroupID == "" {
		r.RemoteGroup.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.securityGroup", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheRemoteGroupID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackSecurityGroup), nil
}

// lookupSecurityGroupIDByName resolves a security-group name to an ID using a
// per-connection cache. Nova reports server security groups by name, but
// Neutron is the source of truth for IDs. Rather than issue its own
// groups.List, this primes the cache from the root securityGroups list
// (memoized once per scan via GetSecurityGroups) — o.securityGroups() sets
// SGNameCache as a side effect, so resolving a server's groups by name no
// longer costs a second Neutron list. A non-nil cache map is the "ready"
// signal; the lock single-flights the first fetch and concurrent callers wait.
func lookupSecurityGroupIDByName(runtime *plugin.Runtime, name string) (string, error) {
	c := conn(runtime)

	c.SGNameCacheLock.Lock()
	cached := c.SGNameCache
	c.SGNameCacheLock.Unlock()
	if cached != nil {
		return cached[name], nil
	}

	// GetSecurityGroups() runs o.securityGroups(), which lists Neutron groups
	// once and primes SGNameCache. Don't hold SGNameCacheLock across this call:
	// o.securityGroups() takes the same lock to prime the cache, so holding it
	// here would deadlock.
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return "", err
	}
	list := root.(*mqlOpenstack).GetSecurityGroups()
	if list.Error != nil {
		return "", list.Error
	}

	c.SGNameCacheLock.Lock()
	defer c.SGNameCacheLock.Unlock()
	if c.SGNameCache == nil {
		// Defensive: build the map from the resource list if priming didn't
		// occur (e.g. the list came back via a path that left the cache unset).
		cache := make(map[string]string, len(list.Data))
		for _, raw := range list.Data {
			sg := raw.(*mqlOpenstackSecurityGroup)
			cache[sg.Name.Data] = sg.Id.Data
		}
		c.SGNameCache = cache
	}
	return c.SGNameCache[name], nil
}

// ---- openstack.network.quotaSet ----

func (r *mqlOpenstackNetworkQuotaSet) id() (string, error) {
	return "openstack.network.quotaSet/" + r.ProjectId.Data, nil
}

func (o *mqlOpenstack) networkQuotaSet() (*mqlOpenstackNetworkQuotaSet, error) {
	c := conn(o.MqlRuntime)
	client, err := c.NetworkClient()
	if err != nil {
		if serviceMissing(err) {
			o.NetworkQuotaSet.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	projectId := c.ProjectID()
	q, err := neutronquotas.Get(ctx(), client, projectId).Extract()
	if err != nil {
		if translateGetError(err) == nil {
			o.NetworkQuotaSet.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	res, err := CreateResource(o.MqlRuntime, "openstack.network.quotaSet", map[string]*llx.RawData{
		"__id":              llx.StringData("openstack.network.quotaSet/" + projectId),
		"projectId":         llx.StringData(projectId),
		"network":           llx.IntData(int64(q.Network)),
		"subnet":            llx.IntData(int64(q.Subnet)),
		"port":              llx.IntData(int64(q.Port)),
		"router":            llx.IntData(int64(q.Router)),
		"floatingIp":        llx.IntData(int64(q.FloatingIP)),
		"securityGroup":     llx.IntData(int64(q.SecurityGroup)),
		"securityGroupRule": llx.IntData(int64(q.SecurityGroupRule)),
		"subnetPool":        llx.IntData(int64(q.SubnetPool)),
		"rbacPolicy":        llx.IntData(int64(q.RBACPolicy)),
		"trunk":             llx.IntData(int64(q.Trunk)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackNetworkQuotaSet), nil
}
