// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsConfig) id() (string, error) {
	return "aws.config", nil
}

func (a *mqlAwsConfig) recorders() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getRecorders(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for _, job := range poolOfJobs.Jobs {
		res = append(res, job.Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsConfig) getRecorders(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getRecorders>calling aws with region %s", region)

			svc := conn.ConfigService(region)
			ctx := context.Background()
			res := []any{}

			params := &configservice.DescribeConfigurationRecordersInput{}
			configRecorders, err := svc.DescribeConfigurationRecorders(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			recorderStatusMap, err := a.describeConfigRecorderStatus(svc, region)
			if err != nil {
				return nil, err
			}
			for _, r := range configRecorders.ConfigurationRecorders {
				var recording bool
				var lastStatus string
				name := getName(convert.ToValue(r.Name), region)
				if val, ok := recorderStatusMap[name]; ok {
					recording = val.recording
					lastStatus = val.lastStatus
				}
				var resourceTypesInterface []any
				var allSupported, includeGlobalResourceTypes bool
				if r.RecordingGroup != nil {
					resourceTypesInterface = make([]any, len(r.RecordingGroup.ResourceTypes))
					for i, resourceType := range r.RecordingGroup.ResourceTypes {
						resourceTypesInterface[i] = string(resourceType)
					}
					allSupported = r.RecordingGroup.AllSupported
					includeGlobalResourceTypes = r.RecordingGroup.IncludeGlobalResourceTypes
				}
				mqlRecorder, err := CreateResource(a.MqlRuntime, "aws.config.recorder",
					map[string]*llx.RawData{
						"name":                       llx.StringDataPtr(r.Name),
						"roleArn":                    llx.StringDataPtr(r.RoleARN),
						"allSupported":               llx.BoolData(allSupported),
						"includeGlobalResourceTypes": llx.BoolData(includeGlobalResourceTypes),
						"resourceTypes":              llx.ArrayData(resourceTypesInterface, types.String),
						"recording":                  llx.BoolData(recording),
						"region":                     llx.StringData(region),
						"lastStatus":                 llx.StringData(lastStatus),
					})
				if err != nil {
					return nil, err
				}
				mqlRecorderRes := mqlRecorder.(*mqlAwsConfigRecorder)
				mqlRecorderRes.cacheRoleArn = r.RoleARN
				res = append(res, mqlRecorderRes)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsConfigRecorderInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsConfigRecorder) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsConfig) deliveryChannels() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDeliveryChannels(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}

	for _, job := range poolOfJobs.Jobs {
		res = append(res, job.Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsConfig) getDeliveryChannels(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getDeliveryChannels>calling aws with region %s", region)

			svc := conn.ConfigService(region)
			ctx := context.Background()
			res := []any{}

			deliveryChannelsParams := &configservice.DescribeDeliveryChannelsInput{}
			deliveryChannels, err := svc.DescribeDeliveryChannels(ctx, deliveryChannelsParams)
			if err != nil {
				return nil, err
			}

			for _, channel := range deliveryChannels.DeliveryChannels {
				mqlDeliveryChannel, err := CreateResource(a.MqlRuntime, "aws.config.deliverychannel",
					map[string]*llx.RawData{
						"name":         llx.StringDataPtr(channel.Name),
						"s3BucketName": llx.StringDataPtr(channel.S3BucketName),
						"s3KeyPrefix":  llx.StringDataPtr(channel.S3KeyPrefix),
						"snsTopicARN":  llx.StringDataPtr(channel.SnsTopicARN),
						"region":       llx.StringData(region),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlDeliveryChannel)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func getName(name string, region string) string {
	return name + "/" + region
}

func (a *mqlAwsConfig) describeConfigRecorderStatus(svc *configservice.Client, regionVal string) (map[string]recorder, error) {
	statusMap := make(map[string]recorder)
	ctx := context.Background()

	params := &configservice.DescribeConfigurationRecorderStatusInput{}
	configRecorderStatus, err := svc.DescribeConfigurationRecorderStatus(ctx, params)
	if err != nil {
		return statusMap, err
	}
	for _, r := range configRecorderStatus.ConfigurationRecordersStatus {
		name := getName(convert.ToValue(r.Name), regionVal)
		statusMap[name] = recorder{recording: r.Recording, lastStatus: string(r.LastStatus)}
	}
	return statusMap, nil
}

type recorder struct {
	recording  bool
	lastStatus string
}

func (a *mqlAwsConfigRecorder) id() (string, error) {
	name := a.Name.Data
	region := a.Region.Data
	return getName(name, region), nil
}

func (a *mqlAwsConfig) rules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getRules(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for _, job := range poolOfJobs.Jobs {
		res = append(res, job.Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsConfig) getRules(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getRules>calling aws with region %s", region)

			svc := conn.ConfigService(region)
			ctx := context.Background()
			res := []any{}

			paginator := configservice.NewDescribeConfigRulesPaginator(svc, &configservice.DescribeConfigRulesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, r := range page.ConfigRules {
					jsonSource, err := convert.JsonToDict(r.Source)
					if err != nil {
						return nil, err
					}
					mqlRule, err := CreateResource(a.MqlRuntime, "aws.config.rule",
						map[string]*llx.RawData{
							"arn":         llx.StringDataPtr(r.ConfigRuleArn),
							"name":        llx.StringDataPtr(r.ConfigRuleName),
							"description": llx.StringDataPtr(r.Description),
							"id":          llx.StringDataPtr(r.ConfigRuleId),
							"source":      llx.MapData(jsonSource, types.Any),
							"state":       llx.StringData(string(r.ConfigRuleState)),
							"region":      llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRule)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsConfigRule) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsConfig) aggregators() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getAggregators(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for _, job := range poolOfJobs.Jobs {
		res = append(res, job.Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsConfig) getAggregators(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getAggregators>calling aws with region %s", region)

			svc := conn.ConfigService(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeConfigurationAggregators(ctx, &configservice.DescribeConfigurationAggregatorsInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, agg := range resp.ConfigurationAggregators {
					// Build account aggregation sources
					accountSources := []any{}
					for i, src := range agg.AccountAggregationSources {
						accountIds := make([]any, len(src.AccountIds))
						for j, id := range src.AccountIds {
							accountIds[j] = id
						}
						awsRegions := make([]any, len(src.AwsRegions))
						for j, r := range src.AwsRegions {
							awsRegions[j] = r
						}
						mqlSrc, err := CreateResource(a.MqlRuntime, ResourceAwsConfigAggregatorAccountAggregationSource,
							map[string]*llx.RawData{
								"__id":          llx.StringData(convert.ToValue(agg.ConfigurationAggregatorArn) + "/accountSource/" + fmt.Sprintf("%d", i)),
								"accountIds":    llx.ArrayData(accountIds, types.String),
								"allAwsRegions": llx.BoolData(src.AllAwsRegions),
								"awsRegions":    llx.ArrayData(awsRegions, types.String),
							})
						if err != nil {
							return nil, err
						}
						accountSources = append(accountSources, mqlSrc)
					}

					mqlAgg, err := CreateResource(a.MqlRuntime, "aws.config.aggregator",
						map[string]*llx.RawData{
							"arn":                       llx.StringDataPtr(agg.ConfigurationAggregatorArn),
							"name":                      llx.StringDataPtr(agg.ConfigurationAggregatorName),
							"region":                    llx.StringData(region),
							"accountAggregationSources": llx.ArrayData(accountSources, types.Resource("aws.config.aggregator.accountAggregationSource")),
							"createdAt":                 llx.TimeDataPtr(agg.CreationTime),
							"lastUpdatedAt":             llx.TimeDataPtr(agg.LastUpdatedTime),
						})
					if err != nil {
						return nil, err
					}

					// Cache the org source role ARN for lazy loading
					mqlAggRes := mqlAgg.(*mqlAwsConfigAggregator)
					if agg.OrganizationAggregationSource != nil {
						mqlAggRes.cacheOrgRoleArn = agg.OrganizationAggregationSource.RoleArn
						mqlAggRes.cacheOrgAllAwsRegions = agg.OrganizationAggregationSource.AllAwsRegions
						mqlAggRes.cacheOrgAwsRegions = agg.OrganizationAggregationSource.AwsRegions
					}

					res = append(res, mqlAggRes)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsConfigAggregatorInternal struct {
	cacheOrgRoleArn       *string
	cacheOrgAllAwsRegions bool
	cacheOrgAwsRegions    []string
}

func (a *mqlAwsConfigAggregator) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsConfigAggregator) organizationAggregationSource() (*mqlAwsConfigAggregatorOrganizationAggregationSource, error) {
	if a.cacheOrgRoleArn == nil {
		a.OrganizationAggregationSource.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	awsRegions := make([]any, len(a.cacheOrgAwsRegions))
	for i, r := range a.cacheOrgAwsRegions {
		awsRegions[i] = r
	}

	mqlSrc, err := CreateResource(a.MqlRuntime, ResourceAwsConfigAggregatorOrganizationAggregationSource,
		map[string]*llx.RawData{
			"__id":          llx.StringData(a.Arn.Data + "/orgSource"),
			"allAwsRegions": llx.BoolData(a.cacheOrgAllAwsRegions),
			"awsRegions":    llx.ArrayData(awsRegions, types.String),
		})
	if err != nil {
		return nil, err
	}
	mqlSrcRes := mqlSrc.(*mqlAwsConfigAggregatorOrganizationAggregationSource)
	mqlSrcRes.cacheRoleArn = a.cacheOrgRoleArn
	return mqlSrcRes, nil
}

type mqlAwsConfigAggregatorOrganizationAggregationSourceInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsConfigAggregatorOrganizationAggregationSource) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsConfigAggregatorOrganizationAggregationSource) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsConfigAggregatorAccountAggregationSource) id() (string, error) {
	return a.__id, nil
}
