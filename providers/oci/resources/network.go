// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
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
			log.Debug().Msgf("calling oci with region %s", *region.RegionKey)

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

func initOciNetworkVcn(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.vcn")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	rawVcns := network.GetVcns()
	if rawVcns.Error != nil {
		return nil, nil, rawVcns.Error
	}

	for _, raw := range rawVcns.Data {
		vcn := raw.(*mqlOciNetworkVcn)
		if vcn.Id.Data == idVal {
			return args, vcn, nil
		}
	}

	return nil, nil, errors.New("oci.network.vcn not found: " + idVal)
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
			log.Debug().Msgf("calling oci with region %s", *region.RegionKey)

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
				sl := mqlInstance.(*mqlOciNetworkSecurityList)
				sl.cacheVcnId = stringValue(securityList.VcnId)
				sl.cacheRegion = stringValue(region.RegionKey)
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkSecurityListInternal struct {
	cacheVcnId string
	// cacheRegion is the region key (e.g. "IAD") discovered when the security
	// list was enumerated. Used by discovery to emit per-region platform IDs
	// without re-parsing the OCID.
	cacheRegion string
}

// initOciNetworkSecurityList resolves a single security list from the scan
// asset's PlatformId when policies reference `oci.network.securityList` on a
// discovered oci-network-securitylist asset. Explicit id takes precedence.
func initOciNetworkSecurityList(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	id := ociArgString(args, "id")
	if id == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "network" || parsed.objectType != "securitylist" {
			return args, nil, nil
		}
		id = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)
	list := network.GetSecurityLists()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		sl := raw.(*mqlOciNetworkSecurityList)
		if sl.Id.Data == id {
			return args, sl, nil
		}
	}
	return nil, nil, errors.New("oci.network.securityList not found: " + id)
}

func (o *mqlOciNetworkSecurityList) id() (string, error) {
	return "oci.network.securityList/" + o.Id.Data, nil
}

func (o *mqlOciNetworkSecurityList) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciNetworkSecurityList) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkSecurityList) hasStatelessRules() (bool, error) {
	if o.IngressSecurityRules.Error != nil {
		return false, o.IngressSecurityRules.Error
	}
	if anyRuleStateless(o.IngressSecurityRules.Data, "stateless") {
		return true, nil
	}
	if o.EgressSecurityRules.Error != nil {
		return false, o.EgressSecurityRules.Error
	}
	return anyRuleStateless(o.EgressSecurityRules.Data, "stateless"), nil
}

func resolveOciCompartment(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciCompartment]) (*mqlOciCompartment, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "oci.compartment", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciCompartment), nil
}

func anyRuleStateless(rules []any, key string) bool {
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := m[key].(bool); ok && v {
			return true
		}
	}
	return false
}

func (o *mqlOciNetwork) subnets() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getSubnets(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getSubnets(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci subnets with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			subnets := []core.Subnet{}
			var page *string
			for {
				response, err := svc.ListSubnets(ctx, core.ListSubnetsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				subnets = append(subnets, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range subnets {
				subnet := subnets[i]

				var created *time.Time
				if subnet.TimeCreated != nil {
					created = &subnet.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(subnet.FreeformTags))
				for k, v := range subnet.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{}, len(subnet.DefinedTags))
				for k, v := range subnet.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
					"id":                      llx.StringDataPtr(subnet.Id),
					"name":                    llx.StringDataPtr(subnet.DisplayName),
					"compartmentID":           llx.StringDataPtr(subnet.CompartmentId),
					"availabilityDomain":      llx.StringDataPtr(subnet.AvailabilityDomain),
					"cidrBlock":               llx.StringDataPtr(subnet.CidrBlock),
					"state":                   llx.StringData(string(subnet.LifecycleState)),
					"dnsLabel":                llx.StringDataPtr(subnet.DnsLabel),
					"subnetDomainName":        llx.StringDataPtr(subnet.SubnetDomainName),
					"prohibitPublicIpOnVnic":  llx.BoolDataPtr(subnet.ProhibitPublicIpOnVnic),
					"prohibitInternetIngress": llx.BoolDataPtr(subnet.ProhibitInternetIngress),
					"created":                 llx.TimeDataPtr(created),
					"freeformTags":            llx.MapData(freeformTags, types.String),
					"definedTags":             llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlSub := mqlInstance.(*mqlOciNetworkSubnet)
				mqlSub.cacheVcnId = stringValue(subnet.VcnId)
				mqlSub.cacheRouteTableId = stringValue(subnet.RouteTableId)
				res = append(res, mqlSub)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkSubnetInternal struct {
	cacheVcnId        string
	cacheRouteTableId string
}

func initOciNetworkSubnet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.subnet")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	rawSubnets := network.GetSubnets()
	if rawSubnets.Error != nil {
		return nil, nil, rawSubnets.Error
	}

	for _, raw := range rawSubnets.Data {
		subnet := raw.(*mqlOciNetworkSubnet)
		if subnet.Id.Data == idVal {
			return args, subnet, nil
		}
	}

	return nil, nil, errors.New("oci.network.subnet not found: " + idVal)
}

func (o *mqlOciNetworkSubnet) id() (string, error) {
	return "oci.network.subnet/" + o.Id.Data, nil
}

func (o *mqlOciNetworkSubnet) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

func (o *mqlOciNetwork) networkSecurityGroups() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getNetworkSecurityGroups(conn, list.Data), 5)
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

func (o *mqlOciNetwork) getNetworkSecurityGroups(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci NSGs with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
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
					"state":         llx.StringData(string(nsg.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlNsg := mqlInstance.(*mqlOciNetworkNetworkSecurityGroup)
				mqlNsg.region = regionResource.Id.Data
				mqlNsg.cacheVcnId = stringValue(nsg.VcnId)
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkNetworkSecurityGroupInternal struct {
	region     string
	cacheVcnId string
	fetchLock  sync.Mutex
	fetched    bool
}

func (o *mqlOciNetworkNetworkSecurityGroup) id() (string, error) {
	return "oci.network.networkSecurityGroup/" + o.Id.Data, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

// NSG security rule for serialization to dict
type nsgSecurityRule struct {
	Direction       string            `json:"direction"`
	Protocol        string            `json:"protocol"`
	Description     string            `json:"description,omitempty"`
	Source          string            `json:"source,omitempty"`
	SourceType      string            `json:"sourceType,omitempty"`
	Destination     string            `json:"destination,omitempty"`
	DestinationType string            `json:"destinationType,omitempty"`
	IsStateless     bool              `json:"isStateless"`
	TcpOptions      *core.TcpOptions  `json:"tcpOptions,omitempty"`
	UdpOptions      *core.UdpOptions  `json:"udpOptions,omitempty"`
	IcmpOptions     *core.IcmpOptions `json:"icmpOptions,omitempty"`
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
	if o.fetched {
		return nil, nil, nil
	}
	o.fetchLock.Lock()
	defer o.fetchLock.Unlock()
	if o.fetched {
		return nil, nil, nil
	}

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
			Direction:       string(rule.Direction),
			Protocol:        stringValue(rule.Protocol),
			Description:     stringValue(rule.Description),
			Source:          stringValue(rule.Source),
			SourceType:      string(rule.SourceType),
			Destination:     stringValue(rule.Destination),
			DestinationType: string(rule.DestinationType),
			IsStateless:     boolValue(rule.IsStateless),
			TcpOptions:      rule.TcpOptions,
			UdpOptions:      rule.UdpOptions,
			IcmpOptions:     rule.IcmpOptions,
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

	o.IngressSecurityRules = plugin.TValue[[]any]{Data: ingress, State: plugin.StateIsSet}
	o.EgressSecurityRules = plugin.TValue[[]any]{Data: egress, State: plugin.StateIsSet}
	o.fetched = true

	return ingress, egress, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) ingressSecurityRules() ([]any, error) {
	_, _, err := o.fetchSecurityRules()
	if err != nil {
		return nil, err
	}
	return o.IngressSecurityRules.Data, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) egressSecurityRules() ([]any, error) {
	_, _, err := o.fetchSecurityRules()
	if err != nil {
		return nil, err
	}
	return o.EgressSecurityRules.Data, nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciNetworkNetworkSecurityGroup) hasStatelessRules() (bool, error) {
	ingress, egress, err := o.fetchSecurityRules()
	if err != nil {
		return false, err
	}
	if ingress == nil {
		ingress = o.IngressSecurityRules.Data
	}
	if egress == nil {
		egress = o.EgressSecurityRules.Data
	}
	if anyRuleStateless(ingress, "isStateless") {
		return true, nil
	}
	return anyRuleStateless(egress, "isStateless"), nil
}

func (o *mqlOciNetworkNetworkSecurityGroup) attachedVnics() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	ctx := context.Background()

	networkClient, err := conn.NetworkClient(o.region)
	if err != nil {
		return nil, err
	}

	var attachments []core.NetworkSecurityGroupVnic
	var page *string
	for {
		response, err := networkClient.ListNetworkSecurityGroupVnics(ctx, core.ListNetworkSecurityGroupVnicsRequest{
			NetworkSecurityGroupId: common.String(o.Id.Data),
			Page:                   page,
		})
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(attachments))
	for i := range attachments {
		att := attachments[i]
		if att.VnicId == nil {
			continue
		}
		// OCI has no batch GetVnic API, so each attachment requires a separate call.
		vnicResp, err := networkClient.GetVnic(ctx, core.GetVnicRequest{VnicId: att.VnicId})
		if err != nil {
			log.Debug().Err(err).Msgf("failed to get VNIC %s", *att.VnicId)
			continue
		}
		mqlVnic, err := ociVnicToMql(o.MqlRuntime, vnicResp.Vnic)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlVnic)
	}
	return res, nil
}

func ociVnicToMql(runtime *plugin.Runtime, vnic core.Vnic) (*mqlOciComputeVnic, error) {
	var created *time.Time
	if vnic.TimeCreated != nil {
		created = &vnic.TimeCreated.Time
	}

	freeformTags := make(map[string]interface{}, len(vnic.FreeformTags))
	for k, v := range vnic.FreeformTags {
		freeformTags[k] = v
	}
	definedTags := make(map[string]interface{}, len(vnic.DefinedTags))
	for k, v := range vnic.DefinedTags {
		definedTags[k] = v
	}

	res, err := CreateResource(runtime, "oci.compute.vnic", map[string]*llx.RawData{
		"id":                  llx.StringDataPtr(vnic.Id),
		"name":                llx.StringDataPtr(vnic.DisplayName),
		"compartmentID":       llx.StringDataPtr(vnic.CompartmentId),
		"isPrimary":           llx.BoolDataPtr(vnic.IsPrimary),
		"privateIp":           llx.StringDataPtr(vnic.PrivateIp),
		"publicIp":            llx.StringDataPtr(vnic.PublicIp),
		"macAddress":          llx.StringDataPtr(vnic.MacAddress),
		"hostnameLabel":       llx.StringDataPtr(vnic.HostnameLabel),
		"nsgIds":              llx.ArrayData(convert.SliceAnyToInterface(vnic.NsgIds), types.String),
		"skipSourceDestCheck": llx.BoolDataPtr(vnic.SkipSourceDestCheck),
		"state":               llx.StringData(string(vnic.LifecycleState)),
		"created":             llx.TimeDataPtr(created),
		"freeformTags":        llx.MapData(freeformTags, types.String),
		"definedTags":         llx.MapData(definedTags, types.Any),
	})
	if err != nil {
		return nil, err
	}
	mqlVnic := res.(*mqlOciComputeVnic)
	mqlVnic.cacheSubnetId = stringValue(vnic.SubnetId)
	return mqlVnic, nil
}

// Internet Gateways

func (o *mqlOciNetwork) internetGateways() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getInternetGateways(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getInternetGateways(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci internet gateways with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			igws := []core.InternetGateway{}
			var page *string
			for {
				response, err := svc.ListInternetGateways(ctx, core.ListInternetGatewaysRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				igws = append(igws, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range igws {
				igw := igws[i]

				var created *time.Time
				if igw.TimeCreated != nil {
					created = &igw.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(igw.FreeformTags))
				for k, v := range igw.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{}, len(igw.DefinedTags))
				for k, v := range igw.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.internetGateway", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(igw.Id),
					"name":          llx.StringDataPtr(igw.DisplayName),
					"compartmentID": llx.StringDataPtr(igw.CompartmentId),
					"isEnabled":     llx.BoolDataPtr(igw.IsEnabled),
					"state":         llx.StringData(string(igw.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlIgw := mqlInstance.(*mqlOciNetworkInternetGateway)
				mqlIgw.cacheVcnId = stringValue(igw.VcnId)
				res = append(res, mqlIgw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkInternetGatewayInternal struct {
	cacheVcnId string
}

func (o *mqlOciNetworkInternetGateway) id() (string, error) {
	return "oci.network.internetGateway/" + o.Id.Data, nil
}

func (o *mqlOciNetworkInternetGateway) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

// NAT Gateways

func (o *mqlOciNetwork) natGateways() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getNatGateways(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getNatGateways(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci NAT gateways with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			natGws := []core.NatGateway{}
			var page *string
			for {
				response, err := svc.ListNatGateways(ctx, core.ListNatGatewaysRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				natGws = append(natGws, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range natGws {
				ngw := natGws[i]

				var created *time.Time
				if ngw.TimeCreated != nil {
					created = &ngw.TimeCreated.Time
				}

				freeformTags := make(map[string]interface{}, len(ngw.FreeformTags))
				for k, v := range ngw.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{}, len(ngw.DefinedTags))
				for k, v := range ngw.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.natGateway", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(ngw.Id),
					"name":          llx.StringDataPtr(ngw.DisplayName),
					"compartmentID": llx.StringDataPtr(ngw.CompartmentId),
					"blockTraffic":  llx.BoolDataPtr(ngw.BlockTraffic),
					"natIp":         llx.StringDataPtr(ngw.NatIp),
					"state":         llx.StringData(string(ngw.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlNgw := mqlInstance.(*mqlOciNetworkNatGateway)
				mqlNgw.cacheVcnId = stringValue(ngw.VcnId)
				res = append(res, mqlNgw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkNatGatewayInternal struct {
	cacheVcnId string
}

func (o *mqlOciNetworkNatGateway) id() (string, error) {
	return "oci.network.natGateway/" + o.Id.Data, nil
}

func (o *mqlOciNetworkNatGateway) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

// Route Tables

// routeRule is an OCI route rule for serialization to dict
type routeRule struct {
	// Target network entity OCID
	NetworkEntityId string `json:"networkEntityId"`
	// Destination CIDR block or service CIDR
	Destination string `json:"destination,omitempty"`
	// Type of destination (CIDR_BLOCK, SERVICE_CIDR_BLOCK)
	DestinationType string `json:"destinationType,omitempty"`
	// Description of the route rule
	Description string `json:"description,omitempty"`
	// Route type (STATIC, LOCAL)
	RouteType string `json:"routeType,omitempty"`
}

func (o *mqlOciNetwork) routeTables() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getRouteTables(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getRouteTables(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci route tables with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			rts := []core.RouteTable{}
			var page *string
			for {
				response, err := svc.ListRouteTables(ctx, core.ListRouteTablesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				rts = append(rts, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range rts {
				rt := rts[i]

				var created *time.Time
				if rt.TimeCreated != nil {
					created = &rt.TimeCreated.Time
				}

				rules := make([]routeRule, 0, len(rt.RouteRules))
				for j := range rt.RouteRules {
					r := rt.RouteRules[j]
					rules = append(rules, routeRule{
						NetworkEntityId: stringValue(r.NetworkEntityId),
						Destination:     stringValue(r.Destination),
						DestinationType: string(r.DestinationType),
						Description:     stringValue(r.Description),
						RouteType:       string(r.RouteType),
					})
				}
				routeRules, err := convert.JsonToDictSlice(rules)
				if err != nil {
					return nil, err
				}

				freeformTags := make(map[string]interface{}, len(rt.FreeformTags))
				for k, v := range rt.FreeformTags {
					freeformTags[k] = v
				}

				definedTags := make(map[string]interface{}, len(rt.DefinedTags))
				for k, v := range rt.DefinedTags {
					definedTags[k] = v
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.routeTable", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(rt.Id),
					"name":          llx.StringDataPtr(rt.DisplayName),
					"compartmentID": llx.StringDataPtr(rt.CompartmentId),
					"routeRules":    llx.DictData(routeRules),
					"state":         llx.StringData(string(rt.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"freeformTags":  llx.MapData(freeformTags, types.String),
					"definedTags":   llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlRt := mqlInstance.(*mqlOciNetworkRouteTable)
				mqlRt.cacheVcnId = stringValue(rt.VcnId)
				res = append(res, mqlRt)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkRouteTableInternal struct {
	cacheVcnId string
}

func initOciNetworkRouteTable(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch oci.network.routeTable")
	}
	idVal := args["id"].Value.(string)

	obj, err := CreateResource(runtime, "oci.network", nil)
	if err != nil {
		return nil, nil, err
	}
	network := obj.(*mqlOciNetwork)

	rawRTs := network.GetRouteTables()
	if rawRTs.Error != nil {
		return nil, nil, rawRTs.Error
	}

	for _, raw := range rawRTs.Data {
		rt := raw.(*mqlOciNetworkRouteTable)
		if rt.Id.Data == idVal {
			return args, rt, nil
		}
	}

	return nil, nil, errors.New("oci.network.routeTable not found: " + idVal)
}

func (o *mqlOciNetworkRouteTable) id() (string, error) {
	return "oci.network.routeTable/" + o.Id.Data, nil
}

func (o *mqlOciNetworkRouteTable) vcn() (*mqlOciNetworkVcn, error) {
	if o.cacheVcnId == "" {
		o.Vcn.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlVcn, err := NewResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheVcnId),
	})
	if err != nil {
		return nil, err
	}
	return mqlVcn.(*mqlOciNetworkVcn), nil
}

// Subnet route table reference

func (o *mqlOciNetworkSubnet) routeTable() (*mqlOciNetworkRouteTable, error) {
	if o.cacheRouteTableId == "" {
		o.RouteTable.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlRt, err := NewResource(o.MqlRuntime, "oci.network.routeTable", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheRouteTableId),
	})
	if err != nil {
		return nil, err
	}
	return mqlRt.(*mqlOciNetworkRouteTable), nil
}

func (o *mqlOciNetwork) publicIps() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getPublicIps(conn, list.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciNetwork) getPublicIps(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci public ips with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			// Region-scoped listing covers reserved public IPs (which persist
			// unattached) and the ephemeral public IPs held by regional
			// entities such as NAT gateways. The OCI API requires a separate
			// call per lifetime at REGION scope.
			publicIps := []core.PublicIp{}
			for _, lifetime := range []core.ListPublicIpsLifetimeEnum{
				core.ListPublicIpsLifetimeReserved,
				core.ListPublicIpsLifetimeEphemeral,
			} {
				var page *string
				for {
					response, err := svc.ListPublicIps(ctx, core.ListPublicIpsRequest{
						Scope:         core.ListPublicIpsScopeRegion,
						CompartmentId: common.String(conn.TenantID()),
						Lifetime:      lifetime,
						Page:          page,
					})
					if err != nil {
						return nil, err
					}

					publicIps = append(publicIps, response.Items...)

					if response.OpcNextPage == nil {
						break
					}
					page = response.OpcNextPage
				}
			}

			var res []any
			for i := range publicIps {
				publicIp := publicIps[i]

				var created *time.Time
				if publicIp.TimeCreated != nil {
					created = &publicIp.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.publicIp", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(publicIp.Id),
					"ipAddress":          llx.StringDataPtr(publicIp.IpAddress),
					"name":               llx.StringDataPtr(publicIp.DisplayName),
					"compartmentID":      llx.StringDataPtr(publicIp.CompartmentId),
					"lifetime":           llx.StringData(string(publicIp.Lifetime)),
					"scope":              llx.StringData(string(publicIp.Scope)),
					"assignedEntityType": llx.StringData(string(publicIp.AssignedEntityType)),
					"assignedEntityId":   llx.StringDataPtr(publicIp.AssignedEntityId),
					"availabilityDomain": llx.StringDataPtr(publicIp.AvailabilityDomain),
					"state":              llx.StringData(string(publicIp.LifecycleState)),
					"created":            llx.TimeDataPtr(created),
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

func (o *mqlOciNetworkPublicIp) id() (string, error) {
	return "oci.network.publicIp/" + o.Id.Data, nil
}
