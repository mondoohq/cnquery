// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (e *mqlAwsSsm) id() (string, error) {
	return "aws.ssm", nil
}

func (a *mqlAwsSsm) parameters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getParameters(conn), 5)
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

func (a *mqlAwsSsm) getParameters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ssm instance list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			res := []interface{}{}
			ssmsvc := conn.Ssm(region)
			ctx := context.Background()

			input := &ssm.DescribeParametersInput{
				Filters: []types.ParametersFilter{},
			}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				resp, err := ssmsvc.DescribeParameters(ctx, input)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather ssm information")
				}
				nextToken = resp.NextToken
				if resp.NextToken != nil {
					input.NextToken = nextToken
				}

				for _, param := range resp.Parameters {
					mqlParam, err := CreateResource(a.MqlRuntime, "aws.ssm.parameter",
						map[string]*llx.RawData{
							"allowedPattern":   llx.StringDataPtr(param.AllowedPattern),
							"arn":              llx.StringDataPtr(param.ARN),
							"region":           llx.StringData(region),
							"dataType":         llx.StringDataPtr(param.DataType),
							"description":      llx.StringDataPtr(param.Description),
							"lastModifiedDate": llx.TimeDataPtr(param.LastModifiedDate),
							"name":             llx.StringDataPtr(param.Name),
							"tier":             llx.StringData(string(param.Tier)),
							"type":             llx.StringData(string(param.Type)),
							"version":          llx.IntData(param.Version),
						})
					if err != nil {
						return nil, err
					}
					mqlParam.(*mqlAwsSsmParameter).parameterCache = param
					res = append(res, mqlParam)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSsmParameterInternal struct {
	parameterCache types.ParameterMetadata
	region         string
}

func (a *mqlAwsSsmParameter) kmsKey() (*mqlAwsKmsKey, error) {
	if a.parameterCache.KeyId == nil {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(kmsKeyArnPattern, a.region, conn.AccountId(), convert.ToString(a.parameterCache.KeyId))),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsSsm) instances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getInstances(conn), 5)
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

func (a *mqlAwsSsm) getInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	log.Debug().Msgf("regions being called for ssm instance list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			res := []interface{}{}
			ssmsvc := conn.Ssm(region)
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
					return nil, errors.Wrap(err, "could not gather ssm information")
				}
				nextToken = isssmresp.NextToken
				if isssmresp.NextToken != nil {
					input.NextToken = nextToken
				}
				ssminstances = append(ssminstances, isssmresp.InstanceInformationList...)
			}

			log.Debug().Str("account", conn.AccountId()).Str("region", region).Int("instance count", len(ssminstances)).Msg("found ec2 ssm instances")
			for _, instance := range ssminstances {
				mqlInstance, err := CreateResource(a.MqlRuntime, "aws.ssm.instance",
					map[string]*llx.RawData{
						"instanceId":      llx.StringDataPtr(instance.InstanceId),
						"pingStatus":      llx.StringData(string(instance.PingStatus)),
						"ipAddress":       llx.StringDataPtr(instance.IPAddress),
						"platformName":    llx.StringDataPtr(instance.PlatformName),
						"platformType":    llx.StringData(string(instance.PlatformType)),
						"platformVersion": llx.StringDataPtr(instance.PlatformVersion),
						"region":          llx.StringData(region),
						"arn":             llx.StringData(ssmInstanceArn(conn.AccountId(), region, convert.ToString(instance.InstanceId))),
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

func ssmInstanceArn(account string, region string, id string) string {
	return fmt.Sprintf(ssmInstanceArnPattern, region, account, id)
}

func (a *mqlAwsSsmParameter) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSsmInstance) id() (string, error) {
	return a.Arn.Data, nil
}

const ssmInstanceArnPattern = "arn:aws:ssm:%s:%s:instance/%s"

func initAwsSsmInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ssm instance")
	}

	obj, err := CreateResource(runtime, "aws.ssm", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	ssm := obj.(*mqlAwsSsm)

	rawResources := ssm.GetInstances()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		instance := rawResources.Data[i].(*mqlAwsSsmInstance)

		if instance.Arn.Data == arnVal {
			return args, instance, nil
		}
	}
	return nil, nil, errors.New("ssm instance does not exist")
}

func (a *mqlAwsSsmInstance) tags() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	id := a.InstanceId.Data
	region := a.Region.Data
	ec2svc := conn.Ec2(region)
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
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}
