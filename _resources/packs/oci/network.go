package oci

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/core"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
	oci_provider "go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	corePack "go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOciNetwork) id() (string, error) {
	return "oci.network", nil
}

func (o *mqlOciNetwork) GetVcns() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getVcns(provider), 5)
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

func (s *mqlOciNetwork) getVcns(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.NetworkClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			vcns, err := s.getVcnsForRegion(ctx, svc, provider.TenantID())
			if err != nil {
				return nil, err
			}

			for i := range vcns {
				vcn := vcns[i]

				var created *time.Time
				if vcn.TimeCreated != nil {
					created = &vcn.TimeCreated.Time
				}

				mqlInstance, err := s.MotorRuntime.CreateResource("oci.network.vcn",
					"id", corePack.ToString(vcn.Id),
					"name", corePack.ToString(vcn.DisplayName),
					"created", created,
					"state", string(vcn.LifecycleState),
					"compartmentID", corePack.ToString(vcn.CompartmentId),
					"cidrBlock", corePack.ToString(vcn.CidrBlock),
					"cidrBlocks", corePack.StrSliceToInterface(vcn.CidrBlocks),
				)
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
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.network.vcn/" + id, nil
}

func (o *mqlOciNetwork) GetSecurityLists() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getSecurityLists(provider), 5)
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

func (s *mqlOciNetwork) getSecurityLists(provider *oci_provider.Provider) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.NetworkClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			securityLists, err := s.getSecurityListsForRegion(ctx, svc, provider.TenantID())
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
						Description:     corePack.ToString(rule.Description),
						Stateless:       corePack.ToBool(rule.IsStateless),
						Protocol:        corePack.ToString(rule.Protocol),
						Destination:     corePack.ToString(rule.Destination),
						DestinationType: string(rule.DestinationType),
						TcpOptions:      rule.TcpOptions,
						UdpOptions:      rule.UdpOptions,
						IcmpOptions:     rule.IcmpOptions,
					})
				}
				egress, err := corePack.JsonToDictSlice(egressSecurityRules)
				if err != nil {
					return nil, err
				}

				ingressSecurityRules := []ingressSecurityRule{}
				for j := range securityList.IngressSecurityRules {
					rule := securityList.IngressSecurityRules[j]
					ingressSecurityRules = append(ingressSecurityRules, ingressSecurityRule{
						Description: corePack.ToString(rule.Description),
						Stateless:   corePack.ToBool(rule.IsStateless),
						Protocol:    corePack.ToString(rule.Protocol),
						Source:      corePack.ToString(rule.Source),
						SourceType:  string(rule.SourceType),
						TcpOptions:  rule.TcpOptions,
						UdpOptions:  rule.UdpOptions,
						IcmpOptions: rule.IcmpOptions,
					})
				}
				ingress, err := corePack.JsonToDictSlice(ingressSecurityRules)
				if err != nil {
					return nil, err
				}

				mqlInstance, err := s.MotorRuntime.CreateResource("oci.network.securityList",
					"id", corePack.ToString(securityList.Id),
					"name", corePack.ToString(securityList.DisplayName),
					"created", created,
					"state", string(securityList.LifecycleState),
					"compartmentID", corePack.ToString(securityList.CompartmentId),
					"egressSecurityRules", egress,
					"ingressSecurityRules", ingress,
				)
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
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "oci.network.securityList/" + id, nil
}
