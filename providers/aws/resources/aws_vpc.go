// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	vpctypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsVpc) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAws) vpcs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getVpcs(conn), 5)
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

func (a *mqlAws) getVpcs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("vpc>getVpcs>calling aws with region %s", region)

			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			params := &ec2.DescribeVpcsInput{
				Filters: conn.Filters.General.ToServerSideEc2Filters(),
			}
			paginator := ec2.NewDescribeVpcsPaginator(svc, params)
			for paginator.HasMorePages() {
				vpcs, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, vpc := range vpcs.Vpcs {
					tagsMap := ec2TagsToMap(vpc.Tags)
					if conn.Filters.General.MatchesExcludeTags(tagsMap) {
						log.Debug().Interface("vpc", vpc.VpcId).Msg("excluding vpc due to filters")
						continue
					}

					name := tagsMap["Name"]
					mqlVpc, err := CreateResource(a.MqlRuntime, ResourceAwsVpc,
						map[string]*llx.RawData{
							"arn":                      llx.StringData(fmt.Sprintf(vpcArnPattern, region, conn.AccountId(), convert.ToValue(vpc.VpcId))),
							"cidrBlock":                llx.StringDataPtr(vpc.CidrBlock),
							"dhcpOptionsId":            llx.StringDataPtr(vpc.DhcpOptionsId),
							"id":                       llx.StringDataPtr(vpc.VpcId),
							"instanceTenancy":          llx.StringData(string(vpc.InstanceTenancy)),
							"internetGatewayBlockMode": llx.StringData(string(vpc.BlockPublicAccessStates.InternetGatewayBlockMode)),
							"isDefault":                llx.BoolData(convert.ToValue(vpc.IsDefault)),
							"name":                     llx.StringData(name),
							"region":                   llx.StringData(region),
							"state":                    llx.StringData(string(vpc.State)),
							"tags":                     llx.MapData(toInterfaceMap(ec2TagsToMap(vpc.Tags)), types.String),
						})
					if err != nil {
						log.Error().Msg(err.Error())
						continue
					}
					res = append(res, mqlVpc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsVpcNatgatewayAddress) id() (string, error) {
	return a.AllocationId.Data, nil
}

func (a *mqlAwsVpcNatgateway) id() (string, error) {
	return a.NatGatewayId.Data, nil
}

type mqlAwsVpcNatgatewayInternal struct {
	natGatewayCache vpctypes.NatGateway
	region          string
}

type mqlAwsVpcNatgatewayAddressInternal struct {
	natGatewayAddressCache vpctypes.NatGatewayAddress
	region                 string
}

func (a *mqlAwsVpcNatgateway) vpc() (*mqlAwsVpc, error) {
	if a.natGatewayCache.VpcId != nil {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpc, map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), convert.ToValue(a.natGatewayCache.VpcId)))})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpc), nil
	}
	a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsVpcNatgateway) subnet() (*mqlAwsVpcSubnet, error) {
	if a.natGatewayCache.SubnetId != nil {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToValue(a.natGatewayCache.SubnetId)))})
		if err != nil {
			a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, err
		}
		return res.(*mqlAwsVpcSubnet), nil
	}
	a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsVpcNatgatewayAddress) publicIp() (*mqlAwsEc2Eip, error) {
	if a.natGatewayAddressCache.PublicIp != nil {
		res, err := NewResource(a.MqlRuntime, ResourceAwsEc2Eip, map[string]*llx.RawData{"publicIp": llx.StringDataPtr(a.natGatewayAddressCache.PublicIp), "region": llx.StringData(a.region)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsEc2Eip), nil
	}
	a.PublicIp.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsVpc) natGateways() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	endpoints := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcId))
	params := &ec2.DescribeNatGatewaysInput{Filter: filters}
	paginator := ec2.NewDescribeNatGatewaysPaginator(svc, params)
	for paginator.HasMorePages() {
		natgateways, err := paginator.NextPage(ctx)
		if err != nil {
			a.NatGateways.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, err
		}

		for _, gw := range natgateways.NatGateways {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(gw.Tags)) {
				log.Debug().Interface("nat_gateway", gw.NatGatewayId).Msg("excluding nat gateway due to filters")
				continue
			}

			addresses := []any{}
			for _, address := range gw.NatGatewayAddresses {
				mqlNatGatewayAddress, err := CreateResource(a.MqlRuntime, ResourceAwsVpcNatgatewayAddress,
					map[string]*llx.RawData{
						"allocationId":       llx.StringDataPtr(address.AllocationId),
						"networkInterfaceId": llx.StringDataPtr(address.NetworkInterfaceId),
						"privateIp":          llx.StringDataPtr(address.PrivateIp),
						"isPrimary":          llx.BoolDataPtr(address.IsPrimary),
					})
				if err == nil {
					mqlNatGatewayAddress.(*mqlAwsVpcNatgatewayAddress).natGatewayAddressCache = address
					mqlNatGatewayAddress.(*mqlAwsVpcNatgatewayAddress).region = a.Region.Data
					addresses = append(addresses, mqlNatGatewayAddress)
				} else {
					log.Error().Err(err).Msg("cannot create vpc natgateway address resource")
				}
			}

			args := map[string]*llx.RawData{
				"createdAt":    llx.TimeDataPtr(gw.CreateTime),
				"natGatewayId": llx.StringDataPtr(gw.NatGatewayId),
				"state":        llx.StringData(string(gw.State)),
				"tags":         llx.MapData(toInterfaceMap(ec2TagsToMap(gw.Tags)), types.String),
				"addresses":    llx.ArrayData(addresses, types.Type(ResourceAwsVpcNatgatewayAddress)),
			}

			mqlNatGat, err := CreateResource(a.MqlRuntime, ResourceAwsVpcNatgateway, args)
			if err != nil {
				return nil, err
			}
			mqlNatGat.(*mqlAwsVpcNatgateway).natGatewayCache = gw
			mqlNatGat.(*mqlAwsVpcNatgateway).region = a.Region.Data

			endpoints = append(endpoints, mqlNatGat)
		}
	}
	return endpoints, nil
}

func (a *mqlAwsVpcEndpoint) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsVpc) endpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	endpoints := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcId))
	params := &ec2.DescribeVpcEndpointsInput{Filters: filters}
	paginator := ec2.NewDescribeVpcEndpointsPaginator(svc, params)
	for paginator.HasMorePages() {
		endpointsRes, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, endpoint := range endpointsRes.VpcEndpoints {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(endpoint.Tags)) {
				log.Debug().Interface("vpc_endpoint", endpoint.VpcEndpointId).Msg("excluding vpc endpoint due to filters")
				continue
			}

			var subnetIds []any
			for _, subnet := range endpoint.SubnetIds {
				subnetIds = append(subnetIds, subnet)
			}
			mqlEndpoint, err := CreateResource(a.MqlRuntime, ResourceAwsVpcEndpoint,
				map[string]*llx.RawData{
					"id":                llx.StringData(fmt.Sprintf("%s/%s", a.Region.Data, *endpoint.VpcEndpointId)),
					"policyDocument":    llx.StringDataPtr(endpoint.PolicyDocument),
					"privateDnsEnabled": llx.BoolDataPtr(endpoint.PrivateDnsEnabled),
					"region":            llx.StringData(a.Region.Data),
					"serviceName":       llx.StringDataPtr(endpoint.ServiceName),
					"state":             llx.StringData(string(endpoint.State)),
					"subnets":           llx.ArrayData(subnetIds, types.String),
					"type":              llx.StringData(string(endpoint.VpcEndpointType)),
					"vpc":               llx.StringDataPtr(endpoint.VpcId),
					"createdAt":         llx.TimeDataPtr(endpoint.CreationTimestamp),
				},
			)
			if err != nil {
				return nil, err
			}
			endpoints = append(endpoints, mqlEndpoint)
		}
	}
	return endpoints, nil
}

func (a *mqlAwsVpcServiceEndpoint) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsVpc) serviceEndpoints() ([]any, error) {
	var (
		conn      = a.MqlRuntime.Connection.(*connection.AwsConnection)
		vpcID     = a.Id.Data
		svc       = conn.Ec2(a.Region.Data)
		ctx       = context.Background()
		endpoints = []any{}
	)

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcID))
	paginator := ec2.NewDescribeVpcEndpointsPaginator(svc, &ec2.DescribeVpcEndpointsInput{Filters: filters})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return endpoints, err
		}

		for _, endpoint := range resp.VpcEndpoints {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(endpoint.Tags)) {
				log.Debug().Interface("vpc_endpoint", endpoint.VpcEndpointId).Msg("excluding vpc endpoint due to filters")
				continue
			}

			dnsNames := convert.Into(endpoint.DnsEntries,
				func(d vpctypes.DnsEntry) any { return convert.ToValue(d.DnsName) },
			)
			mqlEndpoint, err := CreateResource(a.MqlRuntime, ResourceAwsVpcServiceEndpoint,
				map[string]*llx.RawData{
					"id":       llx.StringDataPtr(endpoint.VpcEndpointId),
					"name":     llx.StringDataPtr(endpoint.ServiceName),
					"type":     llx.StringData(string(endpoint.VpcEndpointType)),
					"tags":     llx.MapData(toInterfaceMap(ec2TagsToMap(endpoint.Tags)), types.String),
					"dnsNames": llx.ArrayData(dnsNames, types.String),
					"owner":    llx.StringDataPtr(endpoint.OwnerId),
				},
			)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, mqlEndpoint)

			// store the region for further endpoint info
			cast := mqlEndpoint.(*mqlAwsVpcServiceEndpoint)
			cast.region = a.Region.Data
		}
	}

	return endpoints, nil
}

type mqlAwsVpcServiceEndpointInternal struct {
	region  string
	infoErr error
	lock    sync.Mutex
}

func (a *mqlAwsVpcServiceEndpoint) gatherVpcServiceEndpointInfo() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.infoErr != nil {
		return a.infoErr
	}

	var (
		conn = a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc  = conn.Ec2(a.region)
		ctx  = context.Background()

		// https: //docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpcEndpointServices.html
		params = &ec2.DescribeVpcEndpointServicesInput{
			Filters: []vpctypes.Filter{
				{
					Name:   aws.String("service-name"),
					Values: []string{a.Name.Data},
				},
				{
					Name:   aws.String("service-type"),
					Values: []string{a.Type.Data},
				},
			},
		}
	)

	endpointsRes, err := svc.DescribeVpcEndpointServices(ctx, params)
	if err != nil {
		return err
	}

	if len(endpointsRes.ServiceDetails) == 0 {
		a.infoErr = fmt.Errorf("no vpc service endpoint information found for %s", a.Name.Data)
		return a.infoErr
	}

	service := endpointsRes.ServiceDetails[0]

	dnsNames := convert.Into(service.PrivateDnsNames,
		func(d vpctypes.PrivateDnsDetails) any {
			return convert.ToValue(d.PrivateDnsName)
		},
	)

	a.AcceptanceRequired = plugin.TValue[bool]{Data: convert.ToValue(service.AcceptanceRequired), State: plugin.StateIsSet}
	a.ManagesVpcEndpoints = plugin.TValue[bool]{Data: convert.ToValue(service.ManagesVpcEndpoints), State: plugin.StateIsSet}
	a.VpcEndpointPolicySupported = plugin.TValue[bool]{Data: convert.ToValue(service.VpcEndpointPolicySupported), State: plugin.StateIsSet}
	a.PayerResponsibility = plugin.TValue[string]{Data: string(service.PayerResponsibility), State: plugin.StateIsSet}
	a.PrivateDnsNameVerificationState = plugin.TValue[string]{Data: string(service.PrivateDnsNameVerificationState), State: plugin.StateIsSet}
	a.AvailabilityZones = plugin.TValue[[]any]{Data: convert.SliceAnyToInterface(service.AvailabilityZones), State: plugin.StateIsSet}
	a.PrivateDnsNames = plugin.TValue[[]any]{Data: dnsNames, State: plugin.StateIsSet}

	return nil
}

func (a *mqlAwsVpcServiceEndpoint) acceptanceRequired() (bool, error) {
	return false, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) managesVpcEndpoints() (bool, error) {
	return false, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) vpcEndpointPolicySupported() (bool, error) {
	return false, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) privateDnsNameVerificationState() (string, error) {
	return "", a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) payerResponsibility() (string, error) {
	return "", a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) availabilityZones() ([]any, error) {
	return nil, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcServiceEndpoint) privateDnsNames() ([]any, error) {
	return nil, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcPeeringConnection) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsVpc) peeringConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	pcs := []any{}
	filterKeyVal := "requester-vpc-info.vpc-id"
	filterKeyVal2 := "accepter-vpc-info.vpc-id"

	params := &ec2.DescribeVpcPeeringConnectionsInput{Filters: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}, {Name: &filterKeyVal2, Values: []string{vpc}}}}
	paginator := ec2.NewDescribeVpcPeeringConnectionsPaginator(svc, params)
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, peerconn := range res.VpcPeeringConnections {
			status := ""
			if peerconn.Status != nil {
				status = *peerconn.Status.Message
			}
			mqlPeerConn, err := CreateResource(a.MqlRuntime, ResourceAwsVpcPeeringConnection,
				map[string]*llx.RawData{
					"expirationTime": llx.TimeDataPtr(peerconn.ExpirationTime),
					"id":             llx.StringDataPtr(peerconn.VpcPeeringConnectionId),
					"status":         llx.StringData(status),
					"tags":           llx.MapData(toInterfaceMap(ec2TagsToMap(peerconn.Tags)), types.String),
				},
			)
			if err != nil {
				return nil, err
			}
			mqlPeerConn.(*mqlAwsVpcPeeringConnection).peeringConnectionCache = peerconn
			mqlPeerConn.(*mqlAwsVpcPeeringConnection).region = a.Region.Data
			pcs = append(pcs, mqlPeerConn)
		}
	}
	return pcs, nil
}

func (a *mqlAwsVpcPeeringConnectionPeeringVpc) id() (string, error) {
	return "", nil
}

type mqlAwsVpcPeeringConnectionInternal struct {
	peeringConnectionCache vpctypes.VpcPeeringConnection
	region                 string
}

func (a *mqlAwsVpcPeeringConnection) acceptorVpc() (*mqlAwsVpcPeeringConnectionPeeringVpc, error) {
	acceptor := a.peeringConnectionCache.AccepterVpcInfo
	ipv4 := []any{}
	for i := range acceptor.CidrBlockSet {
		ipv4 = append(ipv4, *acceptor.CidrBlockSet[i].CidrBlock)
	}
	ipv6 := []any{}
	for i := range acceptor.Ipv6CidrBlockSet {
		ipv6 = append(ipv6, *acceptor.Ipv6CidrBlockSet[i].Ipv6CidrBlock)
	}
	mql, err := CreateResource(a.MqlRuntime, ResourceAwsVpcPeeringConnectionPeeringVpc,
		map[string]*llx.RawData{
			"allowDnsResolutionFromRemoteVpc": llx.BoolDataPtr(acceptor.PeeringOptions.AllowDnsResolutionFromRemoteVpc),
			"ipv4CiderBlocks":                 llx.ArrayData(ipv4, types.String),
			"ipv6CiderBlocks":                 llx.ArrayData(ipv6, types.String),
			"ownerID":                         llx.StringDataPtr(acceptor.OwnerId),
			"region":                          llx.StringData(a.region),
			"vpcId":                           llx.StringDataPtr(acceptor.VpcId),
		},
	)
	if err != nil {
		return nil, err
	}

	return mql.(*mqlAwsVpcPeeringConnectionPeeringVpc), nil
}

func (a *mqlAwsVpcPeeringConnectionPeeringVpc) vpc() (*mqlAwsVpc, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res, err := NewResource(a.MqlRuntime, ResourceAwsVpc, map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.Region.Data, conn.AccountId(), a.VpcId.Data))})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}

func (a *mqlAwsVpcPeeringConnection) requestorVpc() (*mqlAwsVpcPeeringConnectionPeeringVpc, error) {
	acceptor := a.peeringConnectionCache.AccepterVpcInfo
	ipv4 := []any{}
	for i := range acceptor.CidrBlockSet {
		ipv4 = append(ipv4, *acceptor.CidrBlockSet[i].CidrBlock)
	}
	ipv6 := []any{}
	for i := range acceptor.Ipv6CidrBlockSet {
		ipv6 = append(ipv6, *acceptor.Ipv6CidrBlockSet[i].Ipv6CidrBlock)
	}
	mql, err := CreateResource(a.MqlRuntime, ResourceAwsVpcPeeringConnectionPeeringVpc,
		map[string]*llx.RawData{
			"allowDnsResolutionFromRemoteVpc": llx.BoolDataPtr(acceptor.PeeringOptions.AllowDnsResolutionFromRemoteVpc),
			"ipv4CiderBlocks":                 llx.ArrayData(ipv4, types.String),
			"ipv6CiderBlocks":                 llx.ArrayData(ipv6, types.String),
			"ownerID":                         llx.StringDataPtr(acceptor.OwnerId),
			"region":                          llx.StringData(a.region),
			// vpc() aws.vpc // ← We can populate this if the VPC is in this account
			"vpcId": llx.StringDataPtr(acceptor.VpcId),
		},
	)
	if err != nil {
		return nil, err
	}

	return mql.(*mqlAwsVpcPeeringConnectionPeeringVpc), nil
}

func (a *mqlAwsVpc) flowLogs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	flowLogs := []any{}
	filterKeyVal := "resource-id"
	params := &ec2.DescribeFlowLogsInput{Filter: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
	paginator := ec2.NewDescribeFlowLogsPaginator(svc, params)
	for paginator.HasMorePages() {
		flowLogsRes, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, flowLog := range flowLogsRes.FlowLogs {
			mqlFlowLog, err := CreateResource(a.MqlRuntime, ResourceAwsVpcFlowlog,
				map[string]*llx.RawData{
					"createdAt":              llx.TimeDataPtr(flowLog.CreationTime),
					"destination":            llx.StringDataPtr(flowLog.LogDestination),
					"destinationType":        llx.StringData(string(flowLog.LogDestinationType)),
					"deliverLogsStatus":      llx.StringDataPtr(flowLog.DeliverLogsStatus),
					"id":                     llx.StringDataPtr(flowLog.FlowLogId),
					"logFormat":              llx.StringDataPtr(flowLog.LogFormat),
					"maxAggregationInterval": llx.IntDataDefault(flowLog.MaxAggregationInterval, 0),
					"region":                 llx.StringData(a.Region.Data),
					"status":                 llx.StringDataPtr(flowLog.FlowLogStatus),
					"tags":                   llx.MapData(toInterfaceMap(ec2TagsToMap(flowLog.Tags)), types.String),
					"trafficType":            llx.StringData(string(flowLog.TrafficType)),
					"vpc":                    llx.StringData(vpc),
				},
			)
			if err != nil {
				return nil, err
			}
			flowLogs = append(flowLogs, mqlFlowLog)
		}
	}
	return flowLogs, nil
}

func (a *mqlAwsVpcRoutetable) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsVpc) routeTables() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcVal := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	res := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcVal))
	params := &ec2.DescribeRouteTablesInput{Filters: filters}
	paginator := ec2.NewDescribeRouteTablesPaginator(svc, params)
	for paginator.HasMorePages() {
		routeTables, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, routeTable := range routeTables.RouteTables {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(routeTable.Tags)) {
				log.Debug().Interface("route_table", routeTable.RouteTableId).Msg("excluding route table due to filters")
				continue
			}

			dictRoutes, err := convert.JsonToDictSlice(routeTable.Routes)
			if err != nil {
				return nil, err
			}
			mqlRouteTable, err := CreateResource(a.MqlRuntime, ResourceAwsVpcRoutetable,
				map[string]*llx.RawData{
					"arn":    llx.StringData(fmt.Sprintf(routeTableArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(routeTable.RouteTableId))),
					"id":     llx.StringDataPtr(routeTable.RouteTableId),
					"region": llx.StringData(a.Region.Data),
					"routes": llx.ArrayData(dictRoutes, types.Any),
					"tags":   llx.MapData(toInterfaceMap(ec2TagsToMap(routeTable.Tags)), types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRouteTable)
			mqlRouteTable.(*mqlAwsVpcRoutetable).cacheAssociations = routeTable.Associations
		}
	}
	return res, nil
}

type mqlAwsVpcRoutetableInternal struct {
	cacheAssociations []vpctypes.RouteTableAssociation
}

func (a *mqlAwsVpcRoutetable) associations() ([]any, error) {
	res := []any{}
	for _, assoc := range a.cacheAssociations {
		state, err := convert.JsonToDict(assoc.AssociationState)
		if err != nil {
			return nil, err
		}
		mqlAssoc, err := CreateResource(a.MqlRuntime, ResourceAwsVpcRoutetableAssociation, map[string]*llx.RawData{
			"routeTableAssociationId": llx.StringDataPtr(assoc.RouteTableAssociationId),
			"associationsState":       llx.DictData(state),
			"gatewayId":               llx.StringDataPtr(assoc.GatewayId),
			"main":                    llx.BoolDataPtr(assoc.Main),
			"routeTableId":            llx.StringDataPtr(assoc.RouteTableId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAssoc)
		mqlAssoc.(*mqlAwsVpcRoutetableAssociation).cacheSubnetId = assoc.SubnetId
		mqlAssoc.(*mqlAwsVpcRoutetableAssociation).region = a.Region.Data
	}
	return res, nil
}

type mqlAwsVpcRoutetableAssociationInternal struct {
	cacheSubnetId *string
	region        string
}

func (a *mqlAwsVpcRoutetableAssociation) subnet() (*mqlAwsVpcSubnet, error) {
	if a.cacheSubnetId != nil {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToValue(a.cacheSubnetId)))})
		if err != nil {
			a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, err
		}
		return res.(*mqlAwsVpcSubnet), nil
	}
	a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsVpcSubnet) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsVpcSubnetInternal struct {
	cacheVpcId string
}

func (a *mqlAwsVpcSubnet) routeTables() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	subnetId := a.Id.Data
	vpcId := a.cacheVpcId

	svc := conn.Ec2(region)
	ctx := context.Background()
	res := []any{}

	// If we don't have the VPC ID cached, we need to look it up
	if vpcId == "" {
		subnets, err := svc.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{subnetId},
		})
		if err != nil {
			return nil, err
		}
		if len(subnets.Subnets) > 0 {
			vpcId = convert.ToValue(subnets.Subnets[0].VpcId)
		}
	}

	if vpcId == "" {
		return res, nil
	}

	// Get all route tables for this VPC
	params := &ec2.DescribeRouteTablesInput{
		Filters: []vpctypes.Filter{{Name: aws.String("vpc-id"), Values: []string{vpcId}}},
	}

	paginator := ec2.NewDescribeRouteTablesPaginator(svc, params)
	var mainRouteTable *vpctypes.RouteTable
	var explicitRouteTable *vpctypes.RouteTable

	for paginator.HasMorePages() {
		routeTables, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for i := range routeTables.RouteTables {
			routeTable := routeTables.RouteTables[i]
			for _, assoc := range routeTable.Associations {
				// Check if this is the main route table
				if assoc.Main != nil && *assoc.Main {
					mainRouteTable = &routeTable
				}
				// Check if this route table has an explicit association with our subnet
				if assoc.SubnetId != nil && *assoc.SubnetId == subnetId {
					explicitRouteTable = &routeTable
				}
			}
		}
	}

	// Use explicit association if exists, otherwise use main route table
	var routeTableToReturn *vpctypes.RouteTable
	if explicitRouteTable != nil {
		routeTableToReturn = explicitRouteTable
	} else if mainRouteTable != nil {
		routeTableToReturn = mainRouteTable
	}

	if routeTableToReturn != nil {
		dictRoutes, err := convert.JsonToDictSlice(routeTableToReturn.Routes)
		if err != nil {
			return nil, err
		}
		mqlRouteTable, err := CreateResource(a.MqlRuntime, ResourceAwsVpcRoutetable,
			map[string]*llx.RawData{
				"arn":    llx.StringData(fmt.Sprintf(routeTableArnPattern, region, conn.AccountId(), convert.ToValue(routeTableToReturn.RouteTableId))),
				"id":     llx.StringDataPtr(routeTableToReturn.RouteTableId),
				"region": llx.StringData(region),
				"routes": llx.ArrayData(dictRoutes, types.Any),
				"tags":   llx.MapData(toInterfaceMap(ec2TagsToMap(routeTableToReturn.Tags)), types.String),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRouteTable)
		mqlRouteTable.(*mqlAwsVpcRoutetable).cacheAssociations = routeTableToReturn.Associations
	}

	return res, nil
}

func (a *mqlAwsVpc) subnets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcVal := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	res := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcVal))
	params := &ec2.DescribeSubnetsInput{Filters: filters}
	paginator := ec2.NewDescribeSubnetsPaginator(svc, params)
	for paginator.HasMorePages() {
		subnets, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, subnet := range subnets.Subnets {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(subnet.Tags)) {
				log.Debug().Interface("subnet", subnet.SubnetId).Msg("excluding subnet due to filters")
				continue
			}

			tagsMap := ec2TagsToMap(subnet.Tags)
			subnetResource, err := CreateResource(a.MqlRuntime, ResourceAwsVpcSubnet,
				map[string]*llx.RawData{
					"arn":                         llx.StringData(fmt.Sprintf(subnetArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(subnet.SubnetId))),
					"assignIpv6AddressOnCreation": llx.BoolDataPtr(subnet.AssignIpv6AddressOnCreation),
					"availabilityZone":            llx.StringDataPtr(subnet.AvailabilityZone),
					"availableIpAddressCount":     llx.IntDataPtr(subnet.AvailableIpAddressCount),
					"cidrs":                       llx.StringDataPtr(subnet.CidrBlock),
					"defaultForAvailabilityZone":  llx.BoolDataPtr(subnet.DefaultForAz),
					"id":                          llx.StringDataPtr(subnet.SubnetId),
					"internetGatewayBlockMode":    llx.StringData(string(subnet.BlockPublicAccessStates.InternetGatewayBlockMode)),
					"mapPublicIpOnLaunch":         llx.BoolDataPtr(subnet.MapPublicIpOnLaunch),
					"name":                        llx.StringData(tagsMap["Name"]),
					"region":                      llx.StringData(a.Region.Data),
					"state":                       llx.StringData(string(subnet.State)),
					"tags":                        llx.MapData(toInterfaceMap(tagsMap), types.String),
				})
			if err != nil {
				return nil, err
			}
			subnetResource.(*mqlAwsVpcSubnet).cacheVpcId = vpcVal
			res = append(res, subnetResource)
		}
	}
	return res, nil
}

func initAwsVpcSubnet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if args["arn"] == nil && args["id"] == nil {
		return nil, nil, errors.New("id or arn required to fetch aws vpc subnet")
	}

	var arnValue, id, region, subnetId string
	if args["arn"] != nil {
		arnValue = args["arn"].Value.(string)
	}
	if args["region"] != nil {
		region = args["region"].Value.(string)
	}
	if args["id"] != nil {
		id = args["id"].Value.(string)
	}
	if id != "" {
		subnetId = id
	} else if arnValue != "" {
		parsed, err := arn.Parse(arnValue)
		if err == nil {
			split := strings.Split(parsed.Resource, "/")
			if len(split) == 2 {
				subnetId = split[1]
				region = parsed.Region
			}
		}
	}
	if subnetId == "" {
		return nil, nil, errors.New("no subnet id specified")
	}

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(region)
	ctx := context.Background()
	subnets, err := svc.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []vpctypes.Filter{{Name: aws.String("subnet-id"), Values: []string{subnetId}}}})
	if err != nil {
		return nil, nil, err
	}

	if len(subnets.Subnets) > 0 {
		subnet := subnets.Subnets[0]
		tagsMap := ec2TagsToMap(subnet.Tags)
		if arnValue != "" {
			args["arn"] = llx.StringData(arnValue)
		} else {
			args["arn"] = llx.StringData(fmt.Sprintf(subnetArnPattern, region, conn.AccountId(), convert.ToValue(subnet.SubnetId)))
		}
		args["assignIpv6AddressOnCreation"] = llx.BoolDataPtr(subnet.AssignIpv6AddressOnCreation)
		args["availabilityZone"] = llx.StringDataPtr(subnet.AvailabilityZone)
		args["availableIpAddressCount"] = llx.IntDataPtr(subnet.AvailableIpAddressCount)
		args["cidrs"] = llx.StringDataPtr(subnet.CidrBlock)
		args["defaultForAvailabilityZone"] = llx.BoolDataPtr(subnet.DefaultForAz)
		args["id"] = llx.StringDataPtr(subnet.SubnetId)
		args["internetGatewayBlockMode"] = llx.StringData(string(subnet.BlockPublicAccessStates.InternetGatewayBlockMode))
		args["mapPublicIpOnLaunch"] = llx.BoolDataPtr(subnet.MapPublicIpOnLaunch)
		args["name"] = llx.StringData(tagsMap["Name"])
		args["region"] = llx.StringData(region)
		args["state"] = llx.StringData(string(subnet.State))
		args["tags"] = llx.MapData(toInterfaceMap(tagsMap), types.String)
		return args, nil, nil
	}
	return nil, nil, errors.New("subnet not found")
}

func initAwsVpc(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws vpc")
	}

	// load all vpcs
	obj, err := CreateResource(runtime, "aws", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	a := obj.(*mqlAws)

	rawResources := a.GetVpcs()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	var match func(secGroup *mqlAwsVpc) bool

	if args["arn"] != nil {
		arnVal := args["arn"].Value.(string)
		match = func(vol *mqlAwsVpc) bool {
			return vol.Arn.Data == arnVal
		}
	}

	for _, rawResource := range rawResources.Data {
		volume := rawResource.(*mqlAwsVpc)
		if match(volume) {
			return args, volume, nil
		}
	}

	return nil, nil, errors.New("vpc does not exist")
}

func vpcFilter(vpcId string) vpctypes.Filter {
	return vpctypes.Filter{
		Name:   aws.String("vpc-id"),
		Values: []string{vpcId},
	}
}

// Internet Gateway implementation

func (a *mqlAwsVpc) internetGateways() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	igws := []any{}

	filterKeyVal := "attachment.vpc-id"
	params := &ec2.DescribeInternetGatewaysInput{
		Filters: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpcId}}},
	}

	paginator := ec2.NewDescribeInternetGatewaysPaginator(svc, params)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, igw := range resp.InternetGateways {
			attachments, err := convert.JsonToDictSlice(igw.Attachments)
			if err != nil {
				return nil, err
			}

			mqlIgw, err := CreateResource(a.MqlRuntime, ResourceAwsEc2Internetgateway,
				map[string]*llx.RawData{
					"arn":         llx.StringData(fmt.Sprintf(internetGwArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(igw.InternetGatewayId))),
					"id":          llx.StringDataPtr(igw.InternetGatewayId),
					"region":      llx.StringData(a.Region.Data),
					"attachments": llx.ArrayData(attachments, types.Any),
					"tags":        llx.MapData(toInterfaceMap(ec2TagsToMap(igw.Tags)), types.String),
				})
			if err != nil {
				return nil, err
			}
			igws = append(igws, mqlIgw)
		}
	}
	return igws, nil
}

// Security Groups link implementation

func (a *mqlAwsVpc) securityGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	sgs := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcId))
	params := &ec2.DescribeSecurityGroupsInput{Filters: filters}
	paginator := ec2.NewDescribeSecurityGroupsPaginator(svc, params)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, sg := range resp.SecurityGroups {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(sg.Tags)) {
				log.Debug().Interface("security_group", sg.GroupId).Msg("excluding security group due to filters")
				continue
			}

			mqlSg, err := NewResource(a.MqlRuntime, ResourceAwsEc2Securitygroup,
				map[string]*llx.RawData{
					"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(sg.GroupId))),
				})
			if err != nil {
				return nil, err
			}
			sgs = append(sgs, mqlSg)
		}
	}
	return sgs, nil
}

// Network ACLs link implementation

func (a *mqlAwsVpc) networkAcls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	acls := []any{}

	filters := conn.Filters.General.ToServerSideEc2Filters()
	filters = append(filters, vpcFilter(vpcId))
	params := &ec2.DescribeNetworkAclsInput{Filters: filters}
	paginator := ec2.NewDescribeNetworkAclsPaginator(svc, params)

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, acl := range resp.NetworkAcls {
			if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(acl.Tags)) {
				log.Debug().Interface("network_acl", acl.NetworkAclId).Msg("excluding network acl due to filters")
				continue
			}

			mqlAcl, err := NewResource(a.MqlRuntime, ResourceAwsEc2Networkacl,
				map[string]*llx.RawData{
					"arn": llx.StringData(fmt.Sprintf(networkAclArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(acl.NetworkAclId))),
				})
			if err != nil {
				return nil, err
			}
			acls = append(acls, mqlAcl)
		}
	}
	return acls, nil
}

// VPN Gateway implementation

func (a *mqlAwsVpcVpnGateway) id() (string, error) {
	return a.Arn.Data, nil
}

const vpnGatewayArnPattern = "arn:aws:ec2:%s:%s:vpn-gateway/%s"

func (a *mqlAwsVpc) vpnGateways() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcId := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	vgws := []any{}

	filterKeyVal := "attachment.vpc-id"
	params := &ec2.DescribeVpnGatewaysInput{
		Filters: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpcId}}},
	}

	resp, err := svc.DescribeVpnGateways(ctx, params)
	if err != nil {
		return nil, err
	}

	for _, vgw := range resp.VpnGateways {
		attachments, err := convert.JsonToDictSlice(vgw.VpcAttachments)
		if err != nil {
			return nil, err
		}

		var availabilityZone string
		if vgw.AvailabilityZone != nil {
			availabilityZone = *vgw.AvailabilityZone
		}

		var amazonSideAsn int64
		if vgw.AmazonSideAsn != nil {
			amazonSideAsn = *vgw.AmazonSideAsn
		}

		mqlVgw, err := CreateResource(a.MqlRuntime, ResourceAwsVpcVpnGateway,
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(vgw.VpnGatewayId),
				"arn":              llx.StringData(fmt.Sprintf(vpnGatewayArnPattern, a.Region.Data, conn.AccountId(), convert.ToValue(vgw.VpnGatewayId))),
				"region":           llx.StringData(a.Region.Data),
				"state":            llx.StringData(string(vgw.State)),
				"type":             llx.StringData(string(vgw.Type)),
				"amazonSideAsn":    llx.IntData(amazonSideAsn),
				"availabilityZone": llx.StringData(availabilityZone),
				"attachments":      llx.ArrayData(attachments, types.Any),
				"tags":             llx.MapData(toInterfaceMap(ec2TagsToMap(vgw.Tags)), types.String),
			})
		if err != nil {
			return nil, err
		}
		vgws = append(vgws, mqlVgw)
	}
	return vgws, nil
}
