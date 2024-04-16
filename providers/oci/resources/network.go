// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/oci/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (o *mqlOciNetwork) id() (string, error) {
	return "oci.network", nil
}

func (o *mqlOciNetwork) vcns() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getVcns(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := conn.NetworkClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
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

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.vcn", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(vcn.Id),
					"name":          llx.StringDataPtr(vcn.DisplayName),
					"created":       llx.TimeDataPtr(created),
					"state":         llx.StringData(string(vcn.LifecycleState)),
					"compartmentID": llx.StringDataPtr(vcn.CompartmentId),
					"cidrBlock":     llx.StringDataPtr(vcn.CidrBlock),
					"cidrBlocks":    llx.ArrayData(convert.SliceAnyToInterface(vcn.CidrBlocks), types.String),
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

func (o *mqlOciNetwork) securityLists() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getSecurityLists(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := conn.NetworkClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
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

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.network.securityList", map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(securityList.Id),
					"name":                 llx.StringDataPtr(securityList.DisplayName),
					"created":              llx.TimeDataPtr(created),
					"state":                llx.StringData(string(securityList.LifecycleState)),
					"compartmentID":        llx.StringDataPtr(securityList.CompartmentId),
					"egressSecurityRules":  llx.DictData(egress),
					"ingressSecurityRules": llx.DictData(ingress),
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
