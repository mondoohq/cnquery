package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (s *lumiAwsVpc) id() (string, error) {
	return s.Id()
}

func (s *lumiAwsVpcFlowlog) id() (string, error) {
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
				vpcs, err := svc.DescribeVpcsRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}
				nextToken = vpcs.NextToken
				if vpcs.NextToken != nil {
					params.NextToken = nextToken
				}

				for i := range vpcs.Vpcs {
					v := vpcs.Vpcs[i]

					stringState, err := ec2.VpcState.MarshalValue(v.State)
					if err != nil {
						return nil, err
					}

					lumiVpc, err := s.Runtime.CreateResource("aws.vpc",
						"id", toString(v.VpcId),
						"state", stringState,
						"isDefault", toBool(v.IsDefault),
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
	params := &ec2.DescribeFlowLogsInput{Filter: []ec2.Filter{{Name: &filterKeyVal, Values: []string{vpc}}}}
	for nextToken != nil {
		flowLogsRes, err := svc.DescribeFlowLogsRequest(params).Send(ctx)
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
