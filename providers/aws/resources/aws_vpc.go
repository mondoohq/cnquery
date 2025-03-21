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
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsVpc) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAws) vpcs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getVpcs(conn), 5)
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

func (a *mqlAws) getVpcs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for i := range regions {
		regionVal := regions[i]
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("vpc>getVpcs>calling aws with region %s", regionVal)

			svc := conn.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeVpcsInput{}
			for nextToken != nil {
				vpcs, err := svc.DescribeVpcs(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				nextToken = vpcs.NextToken
				if vpcs.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range vpcs.Vpcs {
					v := vpcs.Vpcs[i]
					name := ""
					if v.Tags != nil {
						for _, tag := range v.Tags {
							if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
								name = *tag.Value
								break
							}
						}
					}
					mqlVpc, err := CreateResource(a.MqlRuntime, "aws.vpc",
						map[string]*llx.RawData{
							"arn":                      llx.StringData(fmt.Sprintf(vpcArnPattern, regionVal, conn.AccountId(), convert.ToString(v.VpcId))),
							"cidrBlock":                llx.StringDataPtr(v.CidrBlock),
							"id":                       llx.StringDataPtr(v.VpcId),
							"instanceTenancy":          llx.StringData(string(v.InstanceTenancy)),
							"internetGatewayBlockMode": llx.StringData(string(v.BlockPublicAccessStates.InternetGatewayBlockMode)),
							"isDefault":                llx.BoolData(convert.ToBool(v.IsDefault)),
							"name":                     llx.StringData(name),
							"region":                   llx.StringData(regionVal),
							"state":                    llx.StringData(string(v.State)),
							"tags":                     llx.MapData(Ec2TagsToMap(v.Tags), types.String),
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
		res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.region, conn.AccountId(), convert.ToString(a.natGatewayCache.VpcId)))})
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
		res, err := NewResource(a.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToString(a.natGatewayCache.SubnetId)))})
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
		res, err := NewResource(a.MqlRuntime, "aws.ec2.eip", map[string]*llx.RawData{"publicIp": llx.StringDataPtr(a.natGatewayAddressCache.PublicIp), "region": llx.StringData(a.region)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsEc2Eip), nil
	}
	a.PublicIp.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsVpc) natGateways() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	endpoints := []interface{}{}
	filterKeyVal := "vpc-id"
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeNatGatewaysInput{Filter: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
	for nextToken != nil {
		natgateways, err := svc.DescribeNatGateways(ctx, params)
		if err != nil {
			a.NatGateways.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, err
		}
		nextToken = natgateways.NextToken
		if natgateways.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, gw := range natgateways.NatGateways {
			addresses := []interface{}{}
			for i := range gw.NatGatewayAddresses {
				add := gw.NatGatewayAddresses[i]

				mqlNatGatewayAddress, err := CreateResource(a.MqlRuntime, "aws.vpc.natgateway.address",
					map[string]*llx.RawData{
						"allocationId":       llx.StringDataPtr(add.AllocationId),
						"networkInterfaceId": llx.StringDataPtr(add.NetworkInterfaceId),
						"privateIp":          llx.StringDataPtr(add.PrivateIp),
						"isPrimary":          llx.BoolDataPtr(add.IsPrimary),
					})
				if err == nil {
					mqlNatGatewayAddress.(*mqlAwsVpcNatgatewayAddress).natGatewayAddressCache = add
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
				"tags":         llx.MapData(Ec2TagsToMap(gw.Tags), types.String),
				"addresses":    llx.ArrayData(addresses, "aws.vpc.natgatewayaddress"),
			}

			mqlNatGat, err := CreateResource(a.MqlRuntime, "aws.vpc.natgateway", args)
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

func (a *mqlAwsVpc) endpoints() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	endpoints := []interface{}{}
	filterKeyVal := "vpc-id"
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeVpcEndpointsInput{Filters: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
	for nextToken != nil {
		endpointsRes, err := svc.DescribeVpcEndpoints(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = endpointsRes.NextToken
		if endpointsRes.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, endpoint := range endpointsRes.VpcEndpoints {
			var subnetIds []interface{}
			for _, subnet := range endpoint.SubnetIds {
				subnetIds = append(subnetIds, subnet)
			}
			mqlEndpoint, err := CreateResource(a.MqlRuntime, "aws.vpc.endpoint",
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

func (a *mqlAwsVpc) serviceEndpoints() ([]interface{}, error) {
	var (
		conn      = a.MqlRuntime.Connection.(*connection.AwsConnection)
		vpcID     = a.Id.Data
		svc       = conn.Ec2(a.Region.Data)
		ctx       = context.Background()
		endpoints = []interface{}{}
	)

	paginator := ec2.NewDescribeVpcEndpointsPaginator(svc, &ec2.DescribeVpcEndpointsInput{
		Filters: []vpctypes.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})

	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return endpoints, err
		}

		for _, endpoint := range resp.VpcEndpoints {
			dnsNames := convert.Into(endpoint.DnsEntries,
				func(d vpctypes.DnsEntry) any { return convert.ToString(d.DnsName) },
			)
			mqlEndpoint, err := CreateResource(a.MqlRuntime, "aws.vpc.serviceEndpoint",
				map[string]*llx.RawData{
					"id":       llx.StringDataPtr(endpoint.VpcEndpointId),
					"name":     llx.StringDataPtr(endpoint.ServiceName),
					"type":     llx.StringData(string(endpoint.VpcEndpointType)),
					"tags":     llx.MapData(Ec2TagsToMap(endpoint.Tags), types.String),
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
			return convert.ToString(d.PrivateDnsName)
		},
	)

	a.AcceptanceRequired = plugin.TValue[bool]{Data: convert.ToBool(service.AcceptanceRequired), State: plugin.StateIsSet}
	a.ManagesVpcEndpoints = plugin.TValue[bool]{Data: convert.ToBool(service.ManagesVpcEndpoints), State: plugin.StateIsSet}
	a.VpcEndpointPolicySupported = plugin.TValue[bool]{Data: convert.ToBool(service.VpcEndpointPolicySupported), State: plugin.StateIsSet}
	a.PayerResponsibility = plugin.TValue[string]{Data: string(service.PayerResponsibility), State: plugin.StateIsSet}
	a.PrivateDnsNameVerificationState = plugin.TValue[string]{Data: string(service.PrivateDnsNameVerificationState), State: plugin.StateIsSet}
	a.AvailabilityZones = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(service.AvailabilityZones), State: plugin.StateIsSet}
	a.PrivateDnsNames = plugin.TValue[[]interface{}]{Data: dnsNames, State: plugin.StateIsSet}

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
func (a *mqlAwsVpcServiceEndpoint) availabilityZones() ([]interface{}, error) {
	return nil, a.gatherVpcServiceEndpointInfo()
}
func (a *mqlAwsVpcServiceEndpoint) privateDnsNames() ([]interface{}, error) {
	return nil, a.gatherVpcServiceEndpointInfo()
}

func (a *mqlAwsVpcPeeringConnection) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsVpc) peeringConnections() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	pcs := []interface{}{}
	filterKeyVal := "requester-vpc-info.vpc-id"
	filterKeyVal2 := "accepter-vpc-info.vpc-id"

	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeVpcPeeringConnectionsInput{Filters: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}, {Name: &filterKeyVal2, Values: []string{vpc}}}}
	for nextToken != nil {
		res, err := svc.DescribeVpcPeeringConnections(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = res.NextToken
		if res.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, peerconn := range res.VpcPeeringConnections {
			status := ""
			if peerconn.Status != nil {
				status = *peerconn.Status.Message
			}
			mqlPeerConn, err := CreateResource(a.MqlRuntime, "aws.vpc.peeringConnection",
				map[string]*llx.RawData{
					"expirationTime": llx.TimeDataPtr(peerconn.ExpirationTime),
					"id":             llx.StringDataPtr(peerconn.VpcPeeringConnectionId),
					"status":         llx.StringData(status),
					"tags":           llx.MapData(Ec2TagsToMap(peerconn.Tags), types.String),
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
	ipv4 := []interface{}{}
	for i := range acceptor.CidrBlockSet {
		ipv4 = append(ipv4, *acceptor.CidrBlockSet[i].CidrBlock)
	}
	ipv6 := []interface{}{}
	for i := range acceptor.Ipv6CidrBlockSet {
		ipv6 = append(ipv6, *acceptor.Ipv6CidrBlockSet[i].Ipv6CidrBlock)
	}
	mql, err := CreateResource(a.MqlRuntime, "aws.vpc.peeringConnection.peeringVpc",
		map[string]*llx.RawData{
			"allowDnsResolutionFromRemoteVpc":            llx.BoolDataPtr(acceptor.PeeringOptions.AllowDnsResolutionFromRemoteVpc),
			"allowEgressFromLocalClassicLinkToRemoteVpc": llx.BoolDataPtr(acceptor.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc), // this is deprecated by aws...
			"allowEgressFromLocalVpcToRemoteClassicLink": llx.BoolDataPtr(acceptor.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink), // this is deprecated by aws...
			"ipv4CiderBlocks":                            llx.ArrayData(ipv4, types.String),
			"ipv6CiderBlocks":                            llx.ArrayData(ipv6, types.String),
			"ownerID":                                    llx.StringDataPtr(acceptor.OwnerId),
			"region":                                     llx.StringData(a.region),
			"vpcId":                                      llx.StringDataPtr(acceptor.VpcId),
		},
	)
	if err != nil {
		return nil, err
	}

	return mql.(*mqlAwsVpcPeeringConnectionPeeringVpc), nil
}

func (a *mqlAwsVpcPeeringConnectionPeeringVpc) vpc() (*mqlAwsVpc, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.Region.Data, conn.AccountId(), a.VpcId.Data))})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}

func (a *mqlAwsVpcPeeringConnection) requestorVpc() (*mqlAwsVpcPeeringConnectionPeeringVpc, error) {
	acceptor := a.peeringConnectionCache.AccepterVpcInfo
	ipv4 := []interface{}{}
	for i := range acceptor.CidrBlockSet {
		ipv4 = append(ipv4, *acceptor.CidrBlockSet[i].CidrBlock)
	}
	ipv6 := []interface{}{}
	for i := range acceptor.Ipv6CidrBlockSet {
		ipv6 = append(ipv6, *acceptor.Ipv6CidrBlockSet[i].Ipv6CidrBlock)
	}
	mql, err := CreateResource(a.MqlRuntime, "aws.vpc.peeringConnection.peeringVpc",
		map[string]*llx.RawData{
			"allowDnsResolutionFromRemoteVpc":            llx.BoolDataPtr(acceptor.PeeringOptions.AllowDnsResolutionFromRemoteVpc),
			"allowEgressFromLocalClassicLinkToRemoteVpc": llx.BoolDataPtr(acceptor.PeeringOptions.AllowEgressFromLocalClassicLinkToRemoteVpc), // this is deprecated by aws...
			"allowEgressFromLocalVpcToRemoteClassicLink": llx.BoolDataPtr(acceptor.PeeringOptions.AllowEgressFromLocalVpcToRemoteClassicLink), // this is deprecated by aws...
			"ipv4CiderBlocks":                            llx.ArrayData(ipv4, types.String),
			"ipv6CiderBlocks":                            llx.ArrayData(ipv6, types.String),
			"ownerID":                                    llx.StringDataPtr(acceptor.OwnerId),
			"region":                                     llx.StringData(a.region),
			// vpc() aws.vpc // ← We can populate this if the VPC is in this account
			"vpcId": llx.StringDataPtr(acceptor.VpcId),
		},
	)
	if err != nil {
		return nil, err
	}

	return mql.(*mqlAwsVpcPeeringConnectionPeeringVpc), nil
}

func (a *mqlAwsVpc) flowLogs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpc := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	flowLogs := []interface{}{}
	filterKeyVal := "resource-id"
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeFlowLogsInput{Filter: []vpctypes.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
	for nextToken != nil {
		flowLogsRes, err := svc.DescribeFlowLogs(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = flowLogsRes.NextToken
		if flowLogsRes.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, flowLog := range flowLogsRes.FlowLogs {
			mqlFlowLog, err := CreateResource(a.MqlRuntime, "aws.vpc.flowlog",
				map[string]*llx.RawData{
					"createdAt":              llx.TimeDataPtr(flowLog.CreationTime),
					"destination":            llx.StringDataPtr(flowLog.LogDestination),
					"destinationType":        llx.StringData(string(flowLog.LogDestinationType)),
					"deliverLogsStatus":      llx.StringDataPtr(flowLog.DeliverLogsStatus),
					"id":                     llx.StringDataPtr(flowLog.FlowLogId),
					"maxAggregationInterval": llx.IntDataDefault(flowLog.MaxAggregationInterval, 0),
					"region":                 llx.StringData(a.Region.Data),
					"status":                 llx.StringDataPtr(flowLog.FlowLogStatus),
					"tags":                   llx.MapData(Ec2TagsToMap(flowLog.Tags), types.String),
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
	return a.Id.Data, nil
}

func (a *mqlAwsVpc) routeTables() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcVal := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	res := []interface{}{}

	nextToken := aws.String("no_token_to_start_with")
	filterName := "vpc-id"
	params := &ec2.DescribeRouteTablesInput{Filters: []vpctypes.Filter{{Name: &filterName, Values: []string{vpcVal}}}}
	for nextToken != nil {
		routeTables, err := svc.DescribeRouteTables(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = routeTables.NextToken
		if routeTables.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, routeTable := range routeTables.RouteTables {
			dictRoutes, err := convert.JsonToDictSlice(routeTable.Routes)
			if err != nil {
				return nil, err
			}
			mqlRouteTable, err := CreateResource(a.MqlRuntime, "aws.vpc.routetable",
				map[string]*llx.RawData{
					"id":     llx.StringDataPtr(routeTable.RouteTableId),
					"routes": llx.ArrayData(dictRoutes, types.Any),
					"tags":   llx.MapData(Ec2TagsToMap(routeTable.Tags), types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRouteTable)
			mqlRouteTable.(*mqlAwsVpcRoutetable).cacheAssociations = routeTable.Associations
			mqlRouteTable.(*mqlAwsVpcRoutetable).region = a.Region.Data
		}
	}
	return res, nil
}

type mqlAwsVpcRoutetableInternal struct {
	cacheAssociations []vpctypes.RouteTableAssociation
	region            string
}

func (a *mqlAwsVpcRoutetable) associations() ([]interface{}, error) {
	res := []interface{}{}
	for i := range a.cacheAssociations {
		assoc := a.cacheAssociations[i]
		state, err := convert.JsonToDict(assoc.AssociationState)
		if err != nil {
			return nil, err
		}
		mqlAssoc, err := CreateResource(a.MqlRuntime, "aws.vpc.routetable.association", map[string]*llx.RawData{
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
		mqlAssoc.(*mqlAwsVpcRoutetableAssociation).region = a.region
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
		res, err := NewResource(a.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToString(a.cacheSubnetId)))})
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

func (a *mqlAwsVpc) subnets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	vpcVal := a.Id.Data

	svc := conn.Ec2(a.Region.Data)
	ctx := context.Background()
	res := []interface{}{}

	nextToken := aws.String("no_token_to_start_with")
	filterName := "vpc-id"
	params := &ec2.DescribeSubnetsInput{Filters: []vpctypes.Filter{{Name: &filterName, Values: []string{vpcVal}}}}
	for nextToken != nil {
		subnets, err := svc.DescribeSubnets(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = subnets.NextToken
		if subnets.NextToken != nil {
			params.NextToken = nextToken
		}

		for _, subnet := range subnets.Subnets {
			subnetResource, err := CreateResource(a.MqlRuntime, "aws.vpc.subnet",
				map[string]*llx.RawData{
					"arn":                         llx.StringData(fmt.Sprintf(subnetArnPattern, a.Region.Data, conn.AccountId(), convert.ToString(subnet.SubnetId))),
					"assignIpv6AddressOnCreation": llx.BoolDataPtr(subnet.AssignIpv6AddressOnCreation),
					"availabilityZone":            llx.StringDataPtr(subnet.AvailabilityZone),
					"availableIpAddressCount":     llx.IntDataPtr(subnet.AvailableIpAddressCount),
					"cidrs":                       llx.StringDataPtr(subnet.CidrBlock),
					"defaultForAvailabilityZone":  llx.BoolDataPtr(subnet.DefaultForAz),
					"id":                          llx.StringDataPtr(subnet.SubnetId),
					"internetGatewayBlockMode":    llx.StringData(string(subnet.BlockPublicAccessStates.InternetGatewayBlockMode)),
					"mapPublicIpOnLaunch":         llx.BoolDataPtr(subnet.MapPublicIpOnLaunch),
					"region":                      llx.StringData(a.Region.Data),
					"state":                       llx.StringData(string(subnet.State)),
				})
			if err != nil {
				return nil, err
			}
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
		if arnValue != "" {
			args["arn"] = llx.StringData(arnValue)
		} else {
			args["arn"] = llx.StringData(fmt.Sprintf(subnetArnPattern, region, conn.AccountId(), convert.ToString(subnet.SubnetId)))
		}
		args["assignIpv6AddressOnCreation"] = llx.BoolDataPtr(subnet.AssignIpv6AddressOnCreation)
		args["availabilityZone"] = llx.StringDataPtr(subnet.AvailabilityZone)
		args["cidrs"] = llx.StringDataPtr(subnet.CidrBlock)
		args["defaultForAvailabilityZone"] = llx.BoolDataPtr(subnet.DefaultForAz)
		args["id"] = llx.StringDataPtr(subnet.SubnetId)
		args["mapPublicIpOnLaunch"] = llx.BoolDataPtr(subnet.MapPublicIpOnLaunch)
		args["region"] = llx.StringData(region)
		args["state"] = llx.StringData(string(subnet.State))
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

	for i := range rawResources.Data {
		volume := rawResources.Data[i].(*mqlAwsVpc)
		if match(volume) {
			return args, volume, nil
		}
	}

	return nil, nil, errors.New("vpc does not exist")
}
