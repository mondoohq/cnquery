// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	vpctypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/providers/aws/connection"

	"go.mondoo.com/cnquery/types"
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
			log.Debug().Msgf("calling aws with region %s", regionVal)

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

					mqlVpc, err := CreateResource(a.MqlRuntime, "aws.vpc",
						map[string]*llx.RawData{
							"arn":             llx.StringData(fmt.Sprintf(vpcArnPattern, regionVal, conn.AccountId(), convert.ToString(v.VpcId))),
							"id":              llx.StringData(convert.ToString(v.VpcId)),
							"state":           llx.StringData(string(v.State)),
							"isDefault":       llx.BoolData(convert.ToBool(v.IsDefault)),
							"instanceTenancy": llx.StringData(string(v.InstanceTenancy)),
							"cidrBlock":       llx.StringData(convert.ToString(v.CidrBlock)),
							"region":          llx.StringData(regionVal),
							"tags":            llx.MapData(Ec2TagsToMap(v.Tags), types.String),
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
					"id":     llx.StringData(convert.ToString(flowLog.FlowLogId)),
					"vpc":    llx.StringData(vpc),
					"region": llx.StringData(a.Region.Data),
					"status": llx.StringData(convert.ToString(flowLog.FlowLogStatus)),
					"tags":   llx.MapData(Ec2TagsToMap(flowLog.Tags), types.String),
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
					"id":     llx.StringData(convert.ToString(routeTable.RouteTableId)),
					"routes": llx.ArrayData(dictRoutes, types.Any),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRouteTable)
		}
	}
	return res, nil
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
					"arn":                 llx.StringData(fmt.Sprintf(subnetArnPattern, a.Region.Data, conn.AccountId(), convert.ToString(subnet.SubnetId))),
					"id":                  llx.StringData(convert.ToString(subnet.SubnetId)),
					"cidrs":               llx.StringData(*subnet.CidrBlock),
					"mapPublicIpOnLaunch": llx.BoolData(*subnet.MapPublicIpOnLaunch),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, subnetResource)
		}
	}
	return res, nil
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

	rawResources, err := a.vpcs()
	if err != nil {
		return nil, nil, err
	}

	var match func(secGroup *mqlAwsVpc) bool

	if args["arn"] != nil {
		arnVal := args["arn"].Value.(string)
		match = func(vol *mqlAwsVpc) bool {
			return vol.Arn.Data == arnVal
		}
	}

	for i := range rawResources {
		volume := rawResources[i].(*mqlAwsVpc)
		if match(volume) {
			return args, volume, nil
		}
	}

	return nil, nil, errors.New("vpc does not exist")
}
