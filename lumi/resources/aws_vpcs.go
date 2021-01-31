package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

const (
	vpcArnPattern = "arn:aws:vpc:%s:%s:id/%s"
)

func (s *lumiAwsVpc) id() (string, error) {
	return s.Arn()
}

func (s *lumiAwsVpcFlowlog) id() (string, error) {
	return s.Id()
}

func (s *lumiAwsVpcRoutetable) id() (string, error) {
	return s.Id()
}

func (s *lumiAws) GetVpcs() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getVpcs(), 5)
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

func (s *lumiAws) getVpcs() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
	}
	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Ec2(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &ec2.DescribeVpcsInput{}
			for nextToken != nil {
				vpcs, err := svc.DescribeVpcs(ctx, params)
				if err != nil {
					return nil, err
				}
				nextToken = vpcs.NextToken
				if vpcs.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range vpcs.Vpcs {
					v := vpcs.Vpcs[i]

					lumiVpc, err := s.Runtime.CreateResource("aws.vpc",
						"arn", fmt.Sprintf(vpcArnPattern, regionVal, account.ID, toString(v.VpcId)),
						"id", toString(v.VpcId),
						"state", string(v.State),
						"isDefault", v.IsDefault,
						"region", regionVal,
					)
					if err != nil {
						log.Error().Msg(err.Error())
						return nil, err
					}
					res = append(res, lumiVpc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *lumiAwsVpc) GetFlowLogs() ([]interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
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
	svc := at.Ec2(region)
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
			lumiFlowLog, err := s.Runtime.CreateResource("aws.vpc.flowlog",
				"id", toString(flowLog.FlowLogId),
				"vpc", vpc,
				"region", region,
				"status", toString(flowLog.FlowLogStatus),
			)
			if err != nil {
				return nil, err
			}
			flowLogs = append(flowLogs, lumiFlowLog)
		}
	}
	return flowLogs, nil
}
func (p *lumiAwsVpc) init(args *lumi.Args) (*lumi.Args, AwsVpc, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil && (*args)["id"] == nil {
		return nil, nil, errors.New("arn or id required to fetch aws vpc")
	}

	// load all vpcs
	obj, err := p.Runtime.CreateResource("aws")
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
			lumiVpcArn, err := vpc.Arn()
			if err != nil {
				log.Error().Err(err).Msg("vpc is not properly initialized")
				return false
			}
			return lumiVpcArn == arnVal
		}
	}

	if (*args)["id"] != nil {
		idVal := (*args)["id"].(string)
		match = func(vpc AwsVpc) bool {
			lumiVpcId, err := vpc.Id()
			if err != nil {
				log.Error().Err(err).Msg("vpc is not properly initialized")
				return false
			}
			return lumiVpcId == idVal
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

func (s *lumiAwsVpc) GetRouteTables() ([]interface{}, error) {
	vpcVal, err := s.Id()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Ec2("")
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
			dictRoutes, err := jsonToDictSlice(routeTable.Routes)
			if err != nil {
				return nil, err
			}
			lumiRouteTable, err := s.Runtime.CreateResource("aws.vpc.routetable",
				"id", toString(routeTable.RouteTableId),
				"routes", dictRoutes,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiRouteTable)
		}
	}
	return res, nil
}
