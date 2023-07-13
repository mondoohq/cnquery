package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

const (
	vpcArnPattern = "arn:aws:vpc:%s:%s:id/%s"
)

func (s *mqlAwsVpc) id() (string, error) {
	return s.Arn()
}

func (s *mqlAwsVpcFlowlog) id() (string, error) {
	return s.Id()
}

func (s *mqlAwsVpcRoutetable) id() (string, error) {
	return s.Id()
}

func (s *mqlAws) GetVpcs() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getVpcs(provider), 5)
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

func (s *mqlAws) getVpcs(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Ec2(regionVal)
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

					mqlVpc, err := s.MotorRuntime.CreateResource("aws.vpc",
						"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, core.ToString(v.VpcId)),
						"id", core.ToString(v.VpcId),
						"state", string(v.State),
						"isDefault", core.ToBool(v.IsDefault),
						"region", regionVal,
						"tags", Ec2TagsToMap(v.Tags),
					)
					if err != nil {
						log.Error().Msg(err.Error())
						return nil, err
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

func (s *mqlAwsVpc) GetFlowLogs() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	vpc, err := s.Id()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	svc := provider.Ec2(region)
	ctx := context.Background()
	flowLogs := []interface{}{}
	filterKeyVal := "resource-id"
	nextToken := aws.String("no_token_to_start_with")
	params := &ec2.DescribeFlowLogsInput{Filter: []types.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
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
			mqlFlowLog, err := s.MotorRuntime.CreateResource("aws.vpc.flowlog",
				"id", core.ToString(flowLog.FlowLogId),
				"vpc", vpc,
				"region", region,
				"status", core.ToString(flowLog.FlowLogStatus),
				"tags", Ec2TagsToMap(flowLog.Tags),
			)
			if err != nil {
				return nil, err
			}
			flowLogs = append(flowLogs, mqlFlowLog)
		}
	}
	return flowLogs, nil
}

func (p *mqlAwsVpc) init(args *resources.Args) (*resources.Args, AwsVpc, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil && (*args)["id"] == nil {
		return nil, nil, errors.New("arn or id required to fetch aws vpc")
	}

	// load all vpcs
	obj, err := p.MotorRuntime.CreateResource("aws")
	if err != nil {
		return nil, nil, err
	}
	aws := obj.(Aws)

	rawResources, err := aws.Vpcs()
	if err != nil {
		return nil, nil, err
	}

	var match func(secGroup AwsVpc) bool

	if (*args)["arn"] != nil {
		arnVal := (*args)["arn"].(string)
		match = func(vpc AwsVpc) bool {
			mqlVpcArn, err := vpc.Arn()
			if err != nil {
				log.Error().Err(err).Msg("vpc is not properly initialized")
				return false
			}
			return mqlVpcArn == arnVal
		}
	}

	if (*args)["id"] != nil {
		idVal := (*args)["id"].(string)
		match = func(vpc AwsVpc) bool {
			mqlVpcId, err := vpc.Id()
			if err != nil {
				log.Error().Err(err).Msg("vpc is not properly initialized")
				return false
			}
			return mqlVpcId == idVal
		}
	}

	for i := range rawResources {
		vpc := rawResources[i].(AwsVpc)
		if match(vpc) {
			return args, vpc, nil
		}
	}
	return nil, nil, errors.New("vpc does not exist")
}

func (s *mqlAwsVpc) GetRouteTables() ([]interface{}, error) {
	vpcVal, err := s.Id()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Ec2("")
	ctx := context.Background()
	res := []interface{}{}

	nextToken := aws.String("no_token_to_start_with")
	filterName := "vpc-id"
	params := &ec2.DescribeRouteTablesInput{Filters: []types.Filter{{Name: &filterName, Values: []string{vpcVal}}}}
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
			dictRoutes, err := core.JsonToDictSlice(routeTable.Routes)
			if err != nil {
				return nil, err
			}
			mqlRouteTable, err := s.MotorRuntime.CreateResource("aws.vpc.routetable",
				"id", core.ToString(routeTable.RouteTableId),
				"routes", dictRoutes,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRouteTable)
		}
	}
	return res, nil
}
