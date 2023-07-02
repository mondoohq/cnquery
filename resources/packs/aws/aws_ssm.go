package aws

import (
	"context"
	"fmt"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlAwsSsm) id() (string, error) {
	return "aws.ssm", nil
}

func (s *mqlAwsSsm) GetInstances() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(s.getInstances(provider), 5)
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

func (s *mqlAwsSsm) getInstances(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ssm instance list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			res := []interface{}{}
			ssmsvc := provider.Ssm(region)
			ctx := context.Background()

			input := &ssm.DescribeInstanceInformationInput{
				Filters: []types.InstanceInformationStringFilter{},
			}
			nextToken := aws.String("no_token_to_start_with")
			ssminstances := make([]types.InstanceInformation, 0)
			for nextToken != nil {
				isssmresp, err := ssmsvc.DescribeInstanceInformation(ctx, input)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Join(err, errors.New("could not gather ssm information"))
				}
				nextToken = isssmresp.NextToken
				if isssmresp.NextToken != nil {
					input.NextToken = nextToken
				}
				ssminstances = append(ssminstances, isssmresp.InstanceInformationList...)
			}

			log.Debug().Str("account", account.ID).Str("region", region).Int("instance count", len(ssminstances)).Msg("found ec2 ssm instances")
			for _, instance := range ssminstances {
				mqlInstance, err := s.MotorRuntime.CreateResource("aws.ssm.instance",
					"instanceId", core.ToString(instance.InstanceId),
					"pingStatus", string(instance.PingStatus),
					"ipAddress", core.ToString(instance.IPAddress),
					"platformName", core.ToString(instance.PlatformName),
					"region", region,
					"arn", ssmInstanceArn(account.ID, region, core.ToString(instance.InstanceId)),
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

func ssmInstanceArn(account string, region string, id string) string {
	return fmt.Sprintf(ssmInstanceArnPattern, region, account, id)
}

func (s *mqlAwsSsmInstance) id() (string, error) {
	return s.Arn()
}

const ssmInstanceArnPattern = "arn:aws:ssm:%s:%s:instance/%s"

func (d *mqlAwsSsmInstance) init(args *resources.Args) (*resources.Args, AwsSsmInstance, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ssm instance")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.ssm")
	if err != nil {
		return nil, nil, err
	}
	ssm := obj.(AwsSsm)

	rawResources, err := ssm.Instances()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		instance := rawResources[i].(AwsSsmInstance)
		mqlInstArn, err := instance.Arn()
		if err != nil {
			return nil, nil, errors.New("ssm instance does not exist")
		}
		if mqlInstArn == arnVal {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("ssm instance does not exist")
}

func (s *mqlAwsSsmInstance) GetTags() (map[string]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	id, err := s.InstanceId()
	if err != nil {
		return nil, err
	}
	region, err := s.Region()
	if err != nil {
		return nil, err
	}
	ec2svc := provider.Ec2(region)
	tagresp, err := ec2svc.DescribeTags(context.Background(), &ec2.DescribeTagsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{id},
			},
		},
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not gather ssm instance tag information")
	} else if tagresp != nil {
		return Ec2SSMTagsToMap(tagresp.Tags), nil
	}
	return map[string]interface{}{}, nil
}

func Ec2SSMTagsToMap(tags []ec2types.TagDescription) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}
