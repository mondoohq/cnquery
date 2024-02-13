// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsConfig) id() (string, error) {
	return "aws.config", nil
}

func (a *mqlAwsConfig) recorders() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getRecorders(conn), 5)
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

func (a *mqlAwsConfig) getRecorders(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getRecorders>calling aws with region %s", regionVal)

			svc := conn.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigurationRecordersInput{}
			configRecorders, err := svc.DescribeConfigurationRecorders(ctx, params)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			recorderStatusMap, err := a.describeConfigRecorderStatus(svc, regionVal)
			if err != nil {
				return nil, err
			}
			for _, r := range configRecorders.ConfigurationRecorders {
				var recording bool
				var lastStatus string
				name := getName(convert.ToString(r.Name), regionVal)
				if val, ok := recorderStatusMap[name]; ok {
					recording = val.recording
					lastStatus = val.lastStatus
				}
				mqlRecorder, err := CreateResource(a.MqlRuntime, "aws.config.recorder",
					map[string]*llx.RawData{
						"name":                       llx.StringDataPtr(r.Name),
						"roleArn":                    llx.StringDataPtr(r.RoleARN),
						"allSupported":               llx.BoolData(r.RecordingGroup.AllSupported),
						"includeGlobalResourceTypes": llx.BoolData(r.RecordingGroup.IncludeGlobalResourceTypes),
						"resourceTypes": 							llx.ArrayData(stringSliceInterface, types.String),
						"recording":                  llx.BoolData(recording),
						"region":                     llx.StringData(regionVal),
						"lastStatus":                 llx.StringData(lastStatus),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRecorder)
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
		name := getName(convert.ToString(r.Name), regionVal)
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

func (a *mqlAwsConfig) rules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getRules(conn), 5)
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

func (a *mqlAwsConfig) getRules(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("config>getRules>calling aws with region %s", regionVal)

			svc := conn.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigRulesInput{}
			rules, err := svc.DescribeConfigRules(ctx, params)
			if err != nil {
				return nil, err
			}
			for _, r := range rules.ConfigRules {
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
						"region":      llx.StringData(regionVal),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRule)
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
