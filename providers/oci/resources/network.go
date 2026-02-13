// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciNetwork) id() (string, error) {
	return "oci.network", nil
}

func (o *mqlOciNetwork) vcns() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getVcns(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (s *mqlOciNetwork) getVcnsForRegion(ctx context.Context, networkClient *core.VirtualNetworkClient, compartmentID string) ([]core.Vcn, error) {
	vcns := []core.Vcn{}
	var page *string
	for {
		request := core.ListVcnsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := networkClient.ListVcns(ctx, request)
		if err != nil {
			return nil, err
		}

		vcns = append(vcns, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return vcns, nil
}

func (o *mqlOciNetwork) getVcns(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", region)

			svc, err := conn.NetworkClient(*region.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []any
			vcns, err := o.getVcnsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range vcns {
				vcn := vcns[i]

				var created *time.Time
				if vcn.TimeCreated != nil {
					created = &vcn.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{})
				for k, v := range vcn.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{})
				for k, v := range vcn.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(vcn.Id),
					"name":                  llx.StringDataPtr(vcn.DisplayName),
					"created":               llx.TimeDataPtr(created),
					"state":                 llx.StringData(string(vcn.LifecycleState)),
					"compartmentID":         llx.StringDataPtr(vcn.CompartmentId),
					"cidrBlock":             llx.StringDataPtr(vcn.CidrBlock),
					"cidrBlocks":            llx.ArrayData(convert.SliceAnyToInterface(vcn.CidrBlocks), types.String),
					"vcnDomainName":         llx.StringDataPtr(vcn.VcnDomainName),
					"defaultDhcpOptionsId":  llx.StringDataPtr(vcn.DefaultDhcpOptionsId),
					"defaultRouteTableId":   llx.StringDataPtr(vcn.DefaultRouteTableId),
					"defaultSecurityListId": llx.StringDataPtr(vcn.DefaultSecurityListId),
					"dnsLabel":              llx.StringDataPtr(vcn.DnsLabel),
					"freeformTags":          llx.MapData(freeformTags, types.String),
					"definedTags":           llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciNetworkVcn) id() (string, error) {
	return "oci.network.vcn/" + o.Id.Data, nil
}

func (o *mqlOciNetwork) securityLists() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getSecurityLists(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (s *mqlOciNetwork) getSecurityListsForRegion(ctx context.Context, networkClient *core.VirtualNetworkClient, compartmentID string) ([]core.SecurityList, error) {
	securityLists := []core.SecurityList{}
	var page *string
	for {
		request := core.ListSecurityListsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := networkClient.ListSecurityLists(ctx, request)
		if err != nil {
			return nil, err
		}

		securityLists = append(securityLists, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return securityLists, nil
}

// OCI VCN SecurityList egress rule for allowing outbound IP packets
type egressSecurityRule struct {
	// Description of egress rule
	Description string `json:"description,omitempty"`
	// Indicates if this is a stateless rule
	Stateless bool `json:"stateless,omitempty"`
	// Transport protocol, follows http://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
	Protocol string `json:"protocol,omitempty"`
	// Range of allowed IP addresses
	Destination string `json:"destination,omitempty"`
	// Type of destination
	DestinationType string `json:"destination_type,omitempty"`
	// TCP options
	TcpOptions *core.TcpOptions `json:"tcpOptions,omitempty"`
	// Udp options
	UdpOptions *core.UdpOptions `json:"udpOptions,omitempty"`
	// Icmp options
	IcmpOptions *core.IcmpOptions `json:"icmpOptions,omitempty"`
}

// OCI VCN SecurityList Ingress rule for allowing inbound IP packets
type ingressSecurityRule struct {
	// Description of ingress rule
	Description string `json:"description,omitempty"`
	// Indicates if this is a stateless rule
	Stateless bool `json:"stateless,omitempty"`
	// Transport protocol, follows http://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
	Protocol string `json:"protocol,omitempty"`
	// Range of allowed IP addresses
	Source string `json:"source,omitempty"`
	// Type of source
	SourceType string `json:"source_type,omitempty"`
	// TCP options
	TcpOptions *core.TcpOptions `json:"tcpOptions,omitempty"`
	// Udp options
	UdpOptions *core.UdpOptions `json:"udpOptions,omitempty"`
	// Icmp options
	IcmpOptions *core.IcmpOptions `json:"icmpOptions,omitempty"`
}

func (o *mqlOciNetwork) getSecurityLists(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", region)

			svc, err := conn.NetworkClient(*region.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []any
			securityLists, err := o.getSecurityListsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range securityLists {
				securityList := securityLists[i]

				var created *time.Time
				if securityList.TimeCreated != nil {
					created = &securityList.TimeCreated.Time
				}

				egressSecurityRules := []egressSecurityRule{}
				for j := range securityList.EgressSecurityRules {
					rule := securityList.EgressSecurityRules[j]
					egressSecurityRules = append(egressSecurityRules, egressSecurityRule{
						Description:     stringValue(rule.Description),
						Stateless:       boolValue(rule.IsStateless),
						Protocol:        stringValue(rule.Protocol),
						Destination:     stringValue(rule.Destination),
						DestinationType: string(rule.DestinationType),
						TcpOptions:      rule.TcpOptions,
						UdpOptions:      rule.UdpOptions,
						IcmpOptions:     rule.IcmpOptions,
					})
				}
				egress, err := convert.JsonToDictSlice(egressSecurityRules)
				if err != nil {
					return nil, err
				}

				ingressSecurityRules := []ingressSecurityRule{}
				for j := range securityList.IngressSecurityRules {
					rule := securityList.IngressSecurityRules[j]
					ingressSecurityRules = append(ingressSecurityRules, ingressSecurityRule{
						Description: stringValue(rule.Description),
						Stateless:   boolValue(rule.IsStateless),
						Protocol:    stringValue(rule.Protocol),
						Source:      stringValue(rule.Source),
						SourceType:  string(rule.SourceType),
						TcpOptions:  rule.TcpOptions,
						UdpOptions:  rule.UdpOptions,
						IcmpOptions: rule.IcmpOptions,
					})
				}
				ingress, err := convert.JsonToDictSlice(ingressSecurityRules)
				if err != nil {
					return nil, err
				}

				freeformTags := make(map[string]interface{})
				for k, v := range securityList.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{})
				for k, v := range securityList.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.securityList", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(securityList.Id),
					"name":                 llx.StringDataPtr(securityList.DisplayName),
					"created":              llx.TimeDataPtr(created),
					"state":                llx.StringData(string(securityList.LifecycleState)),
					"compartmentID":        llx.StringDataPtr(securityList.CompartmentId),
					"egressSecurityRules":  llx.DictData(egress),
					"ingressSecurityRules": llx.DictData(ingress),
					"vcnId":                llx.StringDataPtr(securityList.VcnId),
					"freeformTags":         llx.MapData(freeformTags, types.String),
					"definedTags":          llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciNetworkSecurityList) id() (string, error) {
	return "oci.network.securityList/" + o.Id.Data, nil
}

func (o *mqlOciNetwork) networkSecurityGroups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getNetworkSecurityGroups(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getNSGsForRegion(ctx context.Context, networkClient *core.VirtualNetworkClient, compartmentID string) ([]core.NetworkSecurityGroup, error) {
	nsgs := []core.NetworkSecurityGroup{}
	var page *string
	for {
		request := core.ListNetworkSecurityGroupsRequest{
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := networkClient.ListNetworkSecurityGroups(ctx, request)
		if err != nil {
			return nil, err
		}

		nsgs = append(nsgs, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return nsgs, nil
}

func (o *mqlOciNetwork) getNetworkSecurityGroups(conn *connection.OciConnection) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", region)

			svc, err := conn.NetworkClient(*region.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []any
			nsgs, err := o.getNSGsForRegion(ctx, svc, conn.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range nsgs {
				nsg := nsgs[i]

				var created *time.Time
				if nsg.TimeCreated != nil {
					created = &nsg.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{})
				for k, v := range nsg.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{})
				for k, v := range nsg.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(nsg.Id),
					"name":          llx.StringDataPtr(nsg.DisplayName),
					"compartmentID": llx.StringDataPtr(nsg.CompartmentId),
					"vcnId":         llx.StringDataPtr(nsg.VcnId),
					"state":         llx.StringData(string(nsg.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlInstance.(*mqlOciNetworkNetworkSecurityGroup).region = *region.RegionKey
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkNetworkSecurityGroupInternal struct {
	region string
}

func (o *mqlOciNetworkNetworkSecurityGroup) id() (string, error) {
	return "oci.network.networkSecurityGroup/" + o.Id.Data, nil
}

// NSG security rule for serialization to dict
type nsgSecurityRule struct {
	Direction   string            `json:"direction"`
	Protocol    string            `json:"protocol"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source,omitempty"`
	SourceType  string            `json:"sourceType,omitempty"`
	Destination string            `json:"destination,omitempty"`
	DestType    string            `json:"destinationType,omitempty"`
	IsStateless bool              `json:"isStateless"`
	TcpOptions  *core.TcpOptions  `json:"tcpOptions,omitempty"`
	UdpOptions  *core.UdpOptions  `json:"udpOptions,omitempty"`
	IcmpOptions *core.IcmpOptions `json:"icmpOptions,omitempty"`
}

func (o *mqlOciNetworkNetworkSecurityGroup) getRulesForNSG(ctx context.Context, networkClient *core.VirtualNetworkClient, nsgId string) ([]core.SecurityRule, error) {
	rules := []core.SecurityRule{}
	var page *string
	for {
		request := core.ListNetworkSecurityGroupSecurityRulesRequest{
			NetworkSecurityGroupId: common.String(nsgId),
			Page:                   page,
		}

		response, err := networkClient.ListNetworkSecurityGroupSecurityRules(ctx, request)
		if err != nil {
			return nil, err
		}

		rules = append(rules, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return rules, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) fetchSecurityRules() (ingress []any, egress []any, err error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	svc, err := conn.NetworkClient(o.region)
	if err != nil {
		return nil, nil, err
	}

	rules, err := o.getRulesForNSG(ctx, svc, o.Id.Data)
	if err != nil {
		return nil, nil, err
	}

	ingressRules := []nsgSecurityRule{}
	egressRules := []nsgSecurityRule{}

	for i := range rules {
		rule := rules[i]
		r := nsgSecurityRule{
			Direction:   string(rule.Direction),
			Protocol:    stringValue(rule.Protocol),
			Description: stringValue(rule.Description),
			Source:      stringValue(rule.Source),
			SourceType:  string(rule.SourceType),
			Destination: stringValue(rule.Destination),
			DestType:    string(rule.DestinationType),
			IsStateless: boolValue(rule.IsStateless),
			TcpOptions:  rule.TcpOptions,
			UdpOptions:  rule.UdpOptions,
			IcmpOptions: rule.IcmpOptions,
		}

		if rule.Direction == core.SecurityRuleDirectionIngress {
			ingressRules = append(ingressRules, r)
		} else {
			egressRules = append(egressRules, r)
		}
	}

	ingress, err = convert.JsonToDictSlice(ingressRules)
	if err != nil {
		return nil, nil, err
	}

	egress, err = convert.JsonToDictSlice(egressRules)
	if err != nil {
		return nil, nil, err
	}

	return ingress, egress, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) ingressSecurityRules() ([]any, error) {
	ingress, egress, err := o.fetchSecurityRules()
	if err != nil {
		return nil, err
	}

	// Cache the egress rules so the other method doesn't need to re-fetch
	o.EgressSecurityRules = plugin.TValue[[]any]{Data: egress, State: plugin.StateIsSet}

	return ingress, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) egressSecurityRules() ([]any, error) {
	ingress, egress, err := o.fetchSecurityRules()
	if err != nil {
		return nil, err
	}

	// Cache the ingress rules so the other method doesn't need to re-fetch
	o.IngressSecurityRules = plugin.TValue[[]any]{Data: ingress, State: plugin.StateIsSet}

	return egress, nil
}
