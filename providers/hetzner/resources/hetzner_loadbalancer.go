// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/hetzner/connection"
)

type mqlHetznerLoadBalancerInternal struct {
	cacheLocation         *hcloud.Location
	cacheLoadBalancerType *hcloud.LoadBalancerType
	cacheServices         []hcloud.LoadBalancerService
	cacheTargets          []hcloud.LoadBalancerTarget
	cachePrivateNet       []hcloud.LoadBalancerPrivateNet
	cachePublicNet        hcloud.LoadBalancerPublicNet
}

func (r *mqlHetznerLoadBalancer) id() (string, error) {
	return fmt.Sprintf("hetzner.loadBalancer/%d", r.Id.Data), nil
}

func (h *mqlHetzner) loadBalancers() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.LoadBalancer, *hcloud.Response, error) {
		return c.Client().LoadBalancer.List(ctx(), hcloud.LoadBalancerListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, lb := range items {
		res, err := newMqlHetznerLoadBalancer(h.MqlRuntime, lb)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerLoadBalancer(runtime *plugin.Runtime, lb *hcloud.LoadBalancer) (*mqlHetznerLoadBalancer, error) {
	publicNet := map[string]any{
		"enabled": lb.PublicNet.Enabled,
		"ipv4":    ipString(lb.PublicNet.IPv4.IP),
		"ipv6":    ipString(lb.PublicNet.IPv6.IP),
	}

	res, err := CreateResource(runtime, "hetzner.loadBalancer", map[string]*llx.RawData{
		"__id":            llx.StringData(fmt.Sprintf("hetzner.loadBalancer/%d", lb.ID)),
		"id":              llx.IntData(lb.ID),
		"name":            llx.StringData(lb.Name),
		"publicNet":       llx.DictData(publicNet),
		"algorithm":       llx.StringData(string(lb.Algorithm.Type)),
		"protection":      llx.DictData(protectionDict(lb.Protection.Delete)),
		"labels":          labelData(lb.Labels),
		"created":         llx.TimeDataPtr(timePtr(lb.Created)),
		"includedTraffic": llx.IntData(int64(lb.IncludedTraffic)),
		"outgoingTraffic": llx.IntData(int64(lb.OutgoingTraffic)),
		"ingoingTraffic":  llx.IntData(int64(lb.IngoingTraffic)),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerLoadBalancer)
	m.cacheLocation = lb.Location
	m.cacheLoadBalancerType = lb.LoadBalancerType
	m.cacheServices = lb.Services
	m.cacheTargets = lb.Targets
	m.cachePrivateNet = lb.PrivateNet
	m.cachePublicNet = lb.PublicNet
	return m, nil
}

func initHetznerLoadBalancer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		// Fall back to a connected hetzner-loadbalancer asset, whose id
		// the discovery step stamped on the connection options.
		id, ok = connection.AssetID(conn(runtime).Conf, connection.OptionLoadBalancer)
		if !ok {
			return args, nil, nil
		}
	}
	lb, _, err := conn(runtime).Client().LoadBalancer.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if lb == nil {
		return nil, nil, notFoundErr("loadBalancer", id)
	}
	res, err := newMqlHetznerLoadBalancer(runtime, lb)
	return args, res, err
}

func (m *mqlHetznerLoadBalancer) location() (*mqlHetznerLocation, error) {
	return resolveTypedResource(&m.Location, m.cacheLocation, func(loc *hcloud.Location) (*mqlHetznerLocation, error) {
		return newMqlHetznerLocation(m.MqlRuntime, loc)
	})
}

func (m *mqlHetznerLoadBalancer) loadBalancerType() (*mqlHetznerLoadBalancerType, error) {
	return resolveTypedResource(&m.LoadBalancerType, m.cacheLoadBalancerType, func(t *hcloud.LoadBalancerType) (*mqlHetznerLoadBalancerType, error) {
		return newMqlHetznerLoadBalancerType(m.MqlRuntime, t)
	})
}

func (m *mqlHetznerLoadBalancer) privateNet() ([]any, error) {
	out := make([]any, 0, len(m.cachePrivateNet))
	for _, p := range m.cachePrivateNet {
		res, err := newMqlHetznerLoadBalancerPrivateNet(m.MqlRuntime, m.Id.Data, p)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// --- loadBalancer.privateNet sub-resource ---

func (r *mqlHetznerLoadBalancerPrivateNet) id() (string, error) {
	return fmt.Sprintf("hetzner.loadBalancer.privateNet/%d/%d", r.LoadBalancerId.Data, r.NetworkId.Data), nil
}

func newMqlHetznerLoadBalancerPrivateNet(runtime *plugin.Runtime, lbID int64, p hcloud.LoadBalancerPrivateNet) (*mqlHetznerLoadBalancerPrivateNet, error) {
	var networkID int64
	if p.Network != nil {
		networkID = p.Network.ID
	}
	res, err := CreateResource(runtime, "hetzner.loadBalancer.privateNet", map[string]*llx.RawData{
		"__id":           llx.StringData(fmt.Sprintf("hetzner.loadBalancer.privateNet/%d/%d", lbID, networkID)),
		"loadBalancerId": llx.IntData(lbID),
		"networkId":      llx.IntData(networkID),
		"ip":             llx.StringData(ipString(p.IP)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlHetznerLoadBalancerPrivateNet), nil
}

func (m *mqlHetznerLoadBalancerPrivateNet) network() (*mqlHetznerNetwork, error) {
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

func (m *mqlHetznerLoadBalancer) services() ([]any, error) {
	out := make([]any, 0, len(m.cacheServices))
	for _, s := range m.cacheServices {
		res, err := newMqlHetznerLoadBalancerService(m.MqlRuntime, m.Id.Data, s)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (m *mqlHetznerLoadBalancer) targets() ([]any, error) {
	out := make([]any, 0, len(m.cacheTargets))
	for i, t := range m.cacheTargets {
		res, err := newMqlHetznerLoadBalancerTarget(m.MqlRuntime, m.Id.Data, i, t)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// --- service sub-resource ---

type mqlHetznerLoadBalancerServiceInternal struct {
	cacheCertificates []*hcloud.Certificate
}

func (r *mqlHetznerLoadBalancerService) id() (string, error) {
	return fmt.Sprintf("hetzner.loadBalancer/%d/service/%d", r.LoadBalancerId.Data, r.ListenPort.Data), nil
}

// loadBalancerHealthCheckDict renders a service health check as a dict. Port and
// Retries are hcloud `int` fields, so they are widened to int64: the dict-to-
// primitive converter only accepts int64 (a raw int fails serialization when the
// healthCheck field is queried).
func loadBalancerHealthCheckDict(hc hcloud.LoadBalancerServiceHealthCheck) map[string]any {
	out := map[string]any{
		"protocol": string(hc.Protocol),
		"port":     int64(hc.Port),
		"interval": hc.Interval.Seconds(),
		"timeout":  hc.Timeout.Seconds(),
		"retries":  int64(hc.Retries),
	}
	if hc.HTTP != nil {
		out["http"] = map[string]any{
			"domain":      hc.HTTP.Domain,
			"path":        hc.HTTP.Path,
			"response":    hc.HTTP.Response,
			"statusCodes": stringSlice(hc.HTTP.StatusCodes),
			"tls":         hc.HTTP.TLS,
		}
	}
	return out
}

func newMqlHetznerLoadBalancerService(runtime *plugin.Runtime, lbID int64, s hcloud.LoadBalancerService) (*mqlHetznerLoadBalancerService, error) {
	hc := loadBalancerHealthCheckDict(s.HealthCheck)

	http := map[string]any{
		"cookieName":     s.HTTP.CookieName,
		"cookieLifetime": s.HTTP.CookieLifetime.Seconds(),
		"redirectHttp":   s.HTTP.RedirectHTTP,
		"stickySessions": s.HTTP.StickySessions,
	}

	res, err := CreateResource(runtime, "hetzner.loadBalancer.service", map[string]*llx.RawData{
		"__id":            llx.StringData(fmt.Sprintf("hetzner.loadBalancer/%d/service/%d", lbID, s.ListenPort)),
		"loadBalancerId":  llx.IntData(lbID),
		"protocol":        llx.StringData(string(s.Protocol)),
		"listenPort":      llx.IntData(int64(s.ListenPort)),
		"destinationPort": llx.IntData(int64(s.DestinationPort)),
		"proxyProtocol":   llx.BoolData(s.Proxyprotocol),
		"healthCheck":     llx.DictData(hc),
		"http":            llx.DictData(http),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerLoadBalancerService)
	m.cacheCertificates = s.HTTP.Certificates
	return m, nil
}

func (m *mqlHetznerLoadBalancerService) certificates() ([]any, error) {
	out := make([]any, 0, len(m.cacheCertificates))
	for _, c := range m.cacheCertificates {
		ref, err := NewResource(m.MqlRuntime, "hetzner.certificate", map[string]*llx.RawData{
			"id": llx.IntData(c.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

// --- target sub-resource ---

type mqlHetznerLoadBalancerTargetInternal struct {
	cacheServerID int64
	// cacheLabelSelectorServerIDs holds the servers a label_selector target
	// currently resolves to (from the target's nested Targets), so
	// labelSelectorTargets() can surface the effective backend set without a
	// refetch.
	cacheLabelSelectorServerIDs []int64
}

func (r *mqlHetznerLoadBalancerTarget) id() (string, error) {
	key := r.Type.Data
	switch r.Type.Data {
	case "server":
		key = fmt.Sprintf("server/%d", r.cacheServerID)
	case "label_selector":
		key = fmt.Sprintf("label/%s", r.LabelSelector.Data)
	case "ip":
		key = fmt.Sprintf("ip/%s", r.Ip.Data)
	}
	return fmt.Sprintf("hetzner.loadBalancer/%d/target/%s", r.LoadBalancerId.Data, key), nil
}

// loadBalancerHealthStatusDicts renders a target's per-listener health status as
// dicts. ListenPort is an hcloud `int`, widened to int64 so the healthStatus
// []dict field serializes (a raw int fails the dict-to-primitive converter).
func loadBalancerHealthStatusDicts(statuses []hcloud.LoadBalancerTargetHealthStatus) []any {
	out := make([]any, 0, len(statuses))
	for _, hs := range statuses {
		out = append(out, map[string]any{
			"listenPort": int64(hs.ListenPort),
			"status":     string(hs.Status),
		})
	}
	return out
}

func newMqlHetznerLoadBalancerTarget(runtime *plugin.Runtime, lbID int64, idx int, t hcloud.LoadBalancerTarget) (*mqlHetznerLoadBalancerTarget, error) {
	healthStatus := loadBalancerHealthStatusDicts(t.HealthStatus)

	labelSelector := ""
	if t.LabelSelector != nil {
		labelSelector = t.LabelSelector.Selector
	}
	ip := ""
	if t.IP != nil {
		ip = t.IP.IP
	}

	var keyForId string
	var serverID int64
	var labelSelectorServerIDs []int64
	switch t.Type {
	case hcloud.LoadBalancerTargetTypeServer:
		if t.Server != nil && t.Server.Server != nil {
			serverID = t.Server.Server.ID
			keyForId = fmt.Sprintf("server/%d", serverID)
		} else {
			keyForId = fmt.Sprintf("server/%d", idx)
		}
	case hcloud.LoadBalancerTargetTypeLabelSelector:
		keyForId = fmt.Sprintf("label/%s", labelSelector)
		labelSelectorServerIDs = targetServerIDs(t.Targets)
	case hcloud.LoadBalancerTargetTypeIP:
		keyForId = fmt.Sprintf("ip/%s", ip)
	default:
		keyForId = fmt.Sprintf("idx/%d", idx)
	}

	res, err := CreateResource(runtime, "hetzner.loadBalancer.target", map[string]*llx.RawData{
		"__id":           llx.StringData(fmt.Sprintf("hetzner.loadBalancer/%d/target/%s", lbID, keyForId)),
		"loadBalancerId": llx.IntData(lbID),
		"type":           llx.StringData(string(t.Type)),
		"healthStatus":   dictArrayData(healthStatus),
		"usePrivateIp":   llx.BoolData(t.UsePrivateIP),
		"labelSelector":  llx.StringData(labelSelector),
		"ip":             llx.StringData(ip),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerLoadBalancerTarget)
	m.cacheServerID = serverID
	m.cacheLabelSelectorServerIDs = labelSelectorServerIDs
	return m, nil
}

func (m *mqlHetznerLoadBalancerTarget) server() (*mqlHetznerServer, error) {
	if m.cacheServerID == 0 {
		m.Server.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ref, err := NewResource(m.MqlRuntime, "hetzner.server", map[string]*llx.RawData{
		"id": llx.IntData(m.cacheServerID),
	})
	if err != nil {
		return nil, err
	}
	return ref.(*mqlHetznerServer), nil
}

func (m *mqlHetznerLoadBalancerTarget) labelSelectorTargets() ([]any, error) {
	return serverRefs(m.MqlRuntime, m.cacheLabelSelectorServerIDs)
}

// targetServerIDs extracts the deduplicated server IDs a label_selector
// target resolves to. Hetzner nests the resolved server targets under the
// label_selector target's Targets field.
func targetServerIDs(targets []hcloud.LoadBalancerTarget) []int64 {
	var ids []int64
	seen := map[int64]struct{}{}
	for _, t := range targets {
		if t.Server == nil || t.Server.Server == nil {
			continue
		}
		id := t.Server.Server.ID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// servers returns the deduplicated backend servers behind the load balancer,
// combining direct server targets with the servers matched by any
// label_selector targets.
func (m *mqlHetznerLoadBalancer) servers() ([]any, error) {
	var ids []int64
	seen := map[int64]struct{}{}
	add := func(id int64) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, t := range m.cacheTargets {
		switch t.Type {
		case hcloud.LoadBalancerTargetTypeServer:
			if t.Server != nil && t.Server.Server != nil {
				add(t.Server.Server.ID)
			}
		case hcloud.LoadBalancerTargetTypeLabelSelector:
			for _, id := range targetServerIDs(t.Targets) {
				add(id)
			}
		}
	}
	return serverRefs(m.MqlRuntime, ids)
}
