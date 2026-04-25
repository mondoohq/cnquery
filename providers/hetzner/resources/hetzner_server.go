// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type mqlHetznerServerInternal struct {
	cacheServerType     *hcloud.ServerType
	cacheDatacenter     *hcloud.Datacenter
	cacheLocation       *hcloud.Location
	cacheImage          *hcloud.Image
	cacheVolumes        []*hcloud.Volume
	cacheFloatingIPs    []*hcloud.FloatingIP
	cachePrimaryIPv4ID  int64
	cachePrimaryIPv6ID  int64
	cachePlacementGroup *hcloud.PlacementGroup
	cacheISO            *hcloud.ISO
	cachePrivateNet     []hcloud.ServerPrivateNet
	cacheFirewallIDs    []int64
	cacheLoadBalancers  []*hcloud.LoadBalancer
}

func (r *mqlHetznerServer) id() (string, error) {
	return fmt.Sprintf("hetzner.server/%d", r.Id.Data), nil
}

func (h *mqlHetzner) servers() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Server, *hcloud.Response, error) {
		return c.Client().Server.List(ctx(), hcloud.ServerListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, s := range items {
		res, err := newMqlHetznerServer(h.MqlRuntime, s)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerServer(runtime *plugin.Runtime, s *hcloud.Server) (*mqlHetznerServer, error) {
	res, err := CreateResource(runtime, "hetzner.server", map[string]*llx.RawData{
		"__id":            llx.StringData(fmt.Sprintf("hetzner.server/%d", s.ID)),
		"id":              llx.IntData(s.ID),
		"name":            llx.StringData(s.Name),
		"status":          llx.StringData(string(s.Status)),
		"created":         llx.TimeDataPtr(timePtr(s.Created)),
		"backupWindow":    llx.StringData(s.BackupWindow),
		"rescueEnabled":   llx.BoolData(s.RescueEnabled),
		"locked":          llx.BoolData(s.Locked),
		"includedTraffic": llx.IntData(int64(s.IncludedTraffic)),
		"outgoingTraffic": llx.IntData(int64(s.OutgoingTraffic)),
		"ingoingTraffic":  llx.IntData(int64(s.IngoingTraffic)),
		"labels":          labelData(s.Labels),
		"protection":      llx.DictData(protectionDictRebuild(s.Protection.Delete, s.Protection.Rebuild)),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerServer)
	m.cacheServerType = s.ServerType
	m.cacheDatacenter = s.Datacenter
	m.cacheLocation = s.Location
	m.cacheImage = s.Image
	m.cacheVolumes = s.Volumes
	m.cacheFloatingIPs = s.PublicNet.FloatingIPs
	m.cachePrimaryIPv4ID = s.PublicNet.IPv4.ID
	m.cachePrimaryIPv6ID = s.PublicNet.IPv6.ID
	m.cachePlacementGroup = s.PlacementGroup
	m.cacheISO = s.ISO
	m.cachePrivateNet = s.PrivateNet
	for _, fw := range s.PublicNet.Firewalls {
		m.cacheFirewallIDs = append(m.cacheFirewallIDs, fw.Firewall.ID)
	}
	m.cacheLoadBalancers = s.LoadBalancers
	return m, nil
}

func initHetznerServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	s, _, err := conn(runtime).Client().Server.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if s == nil {
		return nil, nil, notFoundErr("server", id)
	}
	res, err := newMqlHetznerServer(runtime, s)
	return args, res, err
}

func (m *mqlHetznerServer) serverType() (*mqlHetznerServerType, error) {
	return resolveTypedResource(&m.ServerType, m.cacheServerType, func(t *hcloud.ServerType) (*mqlHetznerServerType, error) {
		return newMqlHetznerServerType(m.MqlRuntime, t)
	})
}

func (m *mqlHetznerServer) datacenter() (*mqlHetznerDatacenter, error) {
	return resolveTypedResource(&m.Datacenter, m.cacheDatacenter, func(dc *hcloud.Datacenter) (*mqlHetznerDatacenter, error) {
		return newMqlHetznerDatacenter(m.MqlRuntime, dc)
	})
}

func (m *mqlHetznerServer) location() (*mqlHetznerLocation, error) {
	loc := m.cacheLocation
	if loc == nil && m.cacheDatacenter != nil {
		loc = m.cacheDatacenter.Location
	}
	return resolveTypedResource(&m.Location, loc, func(l *hcloud.Location) (*mqlHetznerLocation, error) {
		return newMqlHetznerLocation(m.MqlRuntime, l)
	})
}

func (m *mqlHetznerServer) image() (*mqlHetznerImage, error) {
	return resolveTypedResource(&m.Image, m.cacheImage, func(img *hcloud.Image) (*mqlHetznerImage, error) {
		return newMqlHetznerImage(m.MqlRuntime, img)
	})
}

func (m *mqlHetznerServer) volumes() ([]any, error) {
	out := make([]any, 0, len(m.cacheVolumes))
	for _, v := range m.cacheVolumes {
		// Server.Volumes carries partial Volume objects (just IDs); resolve via init.
		ref, err := NewResource(m.MqlRuntime, "hetzner.volume", map[string]*llx.RawData{
			"id": llx.IntData(v.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func (m *mqlHetznerServer) floatingIps() ([]any, error) {
	out := make([]any, 0, len(m.cacheFloatingIPs))
	for _, f := range m.cacheFloatingIPs {
		ref, err := NewResource(m.MqlRuntime, "hetzner.floatingIp", map[string]*llx.RawData{
			"id": llx.IntData(f.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func (m *mqlHetznerServer) firewalls() ([]any, error) {
	out := make([]any, 0, len(m.cacheFirewallIDs))
	for _, id := range m.cacheFirewallIDs {
		ref, err := NewResource(m.MqlRuntime, "hetzner.firewall", map[string]*llx.RawData{
			"id": llx.IntData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func (m *mqlHetznerServer) loadBalancers() ([]any, error) {
	out := make([]any, 0, len(m.cacheLoadBalancers))
	for _, lb := range m.cacheLoadBalancers {
		ref, err := NewResource(m.MqlRuntime, "hetzner.loadBalancer", map[string]*llx.RawData{
			"id": llx.IntData(lb.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func (m *mqlHetznerServer) primaryIpv4() (*mqlHetznerPrimaryIp, error) {
	return primaryIpRefByID(m.MqlRuntime, &m.PrimaryIpv4, m.cachePrimaryIPv4ID)
}

func (m *mqlHetznerServer) primaryIpv6() (*mqlHetznerPrimaryIp, error) {
	return primaryIpRefByID(m.MqlRuntime, &m.PrimaryIpv6, m.cachePrimaryIPv6ID)
}

func primaryIpRefByID(runtime *plugin.Runtime, field *plugin.TValue[*mqlHetznerPrimaryIp], id int64) (*mqlHetznerPrimaryIp, error) {
	if id == 0 {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ref, err := NewResource(runtime, "hetzner.primaryIp", map[string]*llx.RawData{
		"id": llx.IntData(id),
	})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlHetznerPrimaryIp), nil
}

func (m *mqlHetznerServer) placementGroup() (*mqlHetznerPlacementGroup, error) {
	return resolveTypedResource(&m.PlacementGroup, m.cachePlacementGroup, func(pg *hcloud.PlacementGroup) (*mqlHetznerPlacementGroup, error) {
		return newMqlHetznerPlacementGroup(m.MqlRuntime, pg)
	})
}

func (m *mqlHetznerServer) iso() (*mqlHetznerIso, error) {
	return resolveTypedResource(&m.Iso, m.cacheISO, func(iso *hcloud.ISO) (*mqlHetznerIso, error) {
		return newMqlHetznerIso(m.MqlRuntime, iso)
	})
}

func (m *mqlHetznerServer) privateNet() ([]any, error) {
	out := make([]any, 0, len(m.cachePrivateNet))
	for _, p := range m.cachePrivateNet {
		res, err := newMqlHetznerServerPrivateNet(m.MqlRuntime, m.Id.Data, p)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// --- server.privateNet sub-resource ---

func (r *mqlHetznerServerPrivateNet) id() (string, error) {
	return fmt.Sprintf("hetzner.server.privateNet/%d/%d", r.ServerId.Data, r.NetworkId.Data), nil
}

func newMqlHetznerServerPrivateNet(runtime *plugin.Runtime, serverID int64, p hcloud.ServerPrivateNet) (*mqlHetznerServerPrivateNet, error) {
	var networkID int64
	if p.Network != nil {
		networkID = p.Network.ID
	}
	aliases := make([]any, 0, len(p.Aliases))
	for _, a := range p.Aliases {
		aliases = append(aliases, ipString(a))
	}
	res, err := CreateResource(runtime, "hetzner.server.privateNet", map[string]*llx.RawData{
		"__id":       llx.StringData(fmt.Sprintf("hetzner.server.privateNet/%d/%d", serverID, networkID)),
		"serverId":   llx.IntData(serverID),
		"networkId":  llx.IntData(networkID),
		"ip":         llx.StringData(ipString(p.IP)),
		"aliasIps":   llx.ArrayData(aliases, types.String),
		"macAddress": llx.StringData(p.MACAddress),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerServerPrivateNet), nil
}

func (m *mqlHetznerServerPrivateNet) network() (*mqlHetznerNetwork, error) {
	if m.NetworkId.Data == 0 {
		m.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ref, err := NewResource(m.MqlRuntime, "hetzner.network", map[string]*llx.RawData{
		"id": llx.IntData(m.NetworkId.Data),
	})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlHetznerNetwork), nil
}
