// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
	"go.mondoo.com/mql/v13/types"
)

// ----- VPC NAT gateways -----

type mqlDigitaloceanVpcNatGatewayInternal struct {
	vpcUUIDs []string
}

func (r *mqlDigitalocean) vpcNatGateways() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.VPCNATGatewaysListOptions{ListOptions: godo.ListOptions{PerPage: 200}}
	for {
		gateways, resp, err := client.VPCNATGateways.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for _, g := range gateways {
			if g == nil {
				continue
			}

			ingressVpcs := make([]interface{}, 0, len(g.VPCs))
			vpcUUIDs := make([]string, 0, len(g.VPCs))
			for _, v := range g.VPCs {
				if v == nil {
					continue
				}
				vpcUUIDs = append(vpcUUIDs, v.VpcUUID)
				ingressVpcs = append(ingressVpcs, map[string]interface{}{
					"vpcUuid":        v.VpcUUID,
					"gatewayIp":      v.GatewayIP,
					"defaultGateway": v.DefaultGateway,
				})
			}

			egressIPs := []interface{}{}
			if g.Egresses != nil {
				for _, pg := range g.Egresses.PublicGateways {
					if pg != nil {
						egressIPs = append(egressIPs, pg.IPv4)
					}
				}
			}

			res, err := CreateResource(r.MqlRuntime, "digitalocean.vpcNatGateway", map[string]*llx.RawData{
				"id":                     llx.StringData(g.ID),
				"name":                   llx.StringData(g.Name),
				"type":                   llx.StringData(g.Type),
				"state":                  llx.StringData(g.State),
				"region":                 llx.StringData(g.Region),
				"size":                   llx.IntData(int64(g.Size)),
				"ingressVpcs":            llx.ArrayData(ingressVpcs, types.Dict),
				"egressPublicGatewayIps": llx.ArrayData(egressIPs, "\x02"),
				"udpTimeoutSeconds":      llx.IntData(int64(g.UDPTimeoutSeconds)),
				"icmpTimeoutSeconds":     llx.IntData(int64(g.ICMPTimeoutSeconds)),
				"tcpTimeoutSeconds":      llx.IntData(int64(g.TCPTimeoutSeconds)),
				"projectId":              llx.StringData(g.ProjectID),
				"createdAt":              llx.TimeData(g.CreatedAt),
				"updatedAt":              llx.TimeData(g.UpdatedAt),
			})
			if err != nil {
				return nil, err
			}
			res.(*mqlDigitaloceanVpcNatGateway).vpcUUIDs = vpcUUIDs
			all = append(all, res)
		}
		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = page + 1
	}
	return all, nil
}

func (r *mqlDigitaloceanVpcNatGateway) vpcs() ([]interface{}, error) {
	return vpcRefsByUUIDs(r.MqlRuntime, r.vpcUUIDs)
}

func (r *mqlDigitaloceanVpcNatGateway) project() (*mqlDigitaloceanProject, error) {
	return projectRef(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}

// ----- Reserved IPv6 addresses -----

func (r *mqlDigitaloceanReservedIpV6) id() (string, error) {
	return "digitalocean.reservedIpV6/" + r.Ip.Data, nil
}

func (r *mqlDigitaloceanReservedIpV6) droplet() (*mqlDigitaloceanDroplet, error) {
	if r.DropletId.Data == 0 {
		r.Droplet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return dropletRef(r.MqlRuntime, r.DropletId.Data, &r.Droplet)
}

func (r *mqlDigitalocean) reservedIPv6s() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	client := conn.Client()

	var all []interface{}
	opt := &godo.ListOptions{PerPage: 200}
	for {
		ips, resp, err := client.ReservedIPV6s.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}
		for i := range ips {
			ip := ips[i]
			dropletID := int64(0)
			if ip.Droplet != nil {
				dropletID = int64(ip.Droplet.ID)
			}
			res, err := CreateResource(r.MqlRuntime, "digitalocean.reservedIpV6", map[string]*llx.RawData{
				"ip":         llx.StringData(ip.IP),
				"region":     llx.StringData(ip.RegionSlug),
				"reservedAt": llx.TimeData(ip.ReservedAt),
				"dropletId":  llx.IntData(dropletID),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}
		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = page + 1
	}
	return all, nil
}

// vpcRefsByUUIDs resolves a list of VPC UUIDs to typed digitalocean.vpc
// resources, skipping any that cannot be found.
func vpcRefsByUUIDs(runtime *plugin.Runtime, uuids []string) ([]interface{}, error) {
	if len(uuids) == 0 {
		return []interface{}{}, nil
	}
	parent, err := parentDigitalocean(runtime)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, 0, len(uuids))
	for _, uuid := range uuids {
		if uuid == "" {
			continue
		}
		vpc, err := parent.vpcByID(uuid)
		if err != nil {
			return nil, err
		}
		if vpc != nil {
			out = append(out, vpc)
		}
	}
	return out, nil
}
