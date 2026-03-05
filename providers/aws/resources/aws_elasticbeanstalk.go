// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsElasticbeanstalk) id() (string, error) {
	return "aws.elasticbeanstalk", nil
}

func (a *mqlAwsElasticbeanstalk) applications() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getApplications(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsElasticbeanstalk) getApplications(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticbeanstalk>getApplications>calling aws with region %s", region)

			svc := conn.ElasticBeanstalk(region)
			ctx := context.Background()
			res := []any{}

			resp, err := svc.DescribeApplications(ctx, &elasticbeanstalk.DescribeApplicationsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for _, app := range resp.Applications {
				mqlApp, err := CreateResource(a.MqlRuntime, "aws.elasticbeanstalk.application",
					map[string]*llx.RawData{
						"__id":                   llx.StringDataPtr(app.ApplicationArn),
						"arn":                    llx.StringDataPtr(app.ApplicationArn),
						"name":                   llx.StringDataPtr(app.ApplicationName),
						"region":                 llx.StringData(region),
						"description":            llx.StringDataPtr(app.Description),
						"createdAt":              llx.TimeDataPtr(app.DateCreated),
						"updatedAt":              llx.TimeDataPtr(app.DateUpdated),
						"configurationTemplates": llx.ArrayData(convert.SliceAnyToInterface(app.ConfigurationTemplates), types.String),
						"versions":               llx.ArrayData(convert.SliceAnyToInterface(app.Versions), types.String),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlApp)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElasticbeanstalkApplication) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.ElasticBeanstalk(region)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &elasticbeanstalk.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.ResourceTags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

func (a *mqlAwsElasticbeanstalk) environments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEnvironments(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsElasticbeanstalk) getEnvironments(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticbeanstalk>getEnvironments>calling aws with region %s", region)

			svc := conn.ElasticBeanstalk(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, env := range resp.Environments {
					tier, _ := convert.JsonToDict(env.Tier)
					mqlEnv, err := CreateResource(a.MqlRuntime, "aws.elasticbeanstalk.environment",
						map[string]*llx.RawData{
							"__id":              llx.StringDataPtr(env.EnvironmentArn),
							"arn":               llx.StringDataPtr(env.EnvironmentArn),
							"name":              llx.StringDataPtr(env.EnvironmentName),
							"region":            llx.StringData(region),
							"applicationName":   llx.StringDataPtr(env.ApplicationName),
							"description":       llx.StringDataPtr(env.Description),
							"environmentId":     llx.StringDataPtr(env.EnvironmentId),
							"platformArn":       llx.StringDataPtr(env.PlatformArn),
							"solutionStackName": llx.StringDataPtr(env.SolutionStackName),
							"status":            llx.StringData(string(env.Status)),
							"health":            llx.StringData(string(env.Health)),
							"healthStatus":      llx.StringData(string(env.HealthStatus)),
							"cname":             llx.StringDataPtr(env.CNAME),
							"endpointUrl":       llx.StringDataPtr(env.EndpointURL),
							"tier":              llx.DictData(tier),
							"createdAt":         llx.TimeDataPtr(env.DateCreated),
							"updatedAt":         llx.TimeDataPtr(env.DateUpdated),
							"versionLabel":      llx.StringDataPtr(env.VersionLabel),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEnv)
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

func (a *mqlAwsElasticbeanstalkEnvironment) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.ElasticBeanstalk(region)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &elasticbeanstalk.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.ResourceTags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}
