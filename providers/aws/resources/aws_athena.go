// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	athena_types "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsAthena) id() (string, error) {
	return "aws.athena", nil
}

func (a *mqlAwsAthena) workgroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getWorkgroups(conn), 5)
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

func (a *mqlAwsAthena) getWorkgroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("athena>getWorkgroups>calling aws with region %s", region)

			svc := conn.Athena(region)
			ctx := context.Background()
			res := []any{}

			paginator := athena.NewListWorkGroupsPaginator(svc, &athena.ListWorkGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, wgSummary := range page.WorkGroups {
					// GetWorkGroup provides full configuration details
					wgResp, err := svc.GetWorkGroup(ctx, &athena.GetWorkGroupInput{
						WorkGroup: wgSummary.Name,
					})
					if err != nil {
						var nfe *athena_types.ResourceNotFoundException
						if errors.As(err, &nfe) {
							log.Warn().Str("workgroup", convert.ToValue(wgSummary.Name)).Msg("workgroup not found, skipping")
							continue
						}
						return nil, err
					}
					mqlWg, err := newMqlAwsAthenaWorkgroup(a.MqlRuntime, region, conn.AccountId(), wgResp.WorkGroup)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlWg)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsAthenaWorkgroup(runtime *plugin.Runtime, region string, accountID string, wg *athena_types.WorkGroup) (*mqlAwsAthenaWorkgroup, error) {
	if wg == nil {
		return nil, fmt.Errorf("workgroup is nil")
	}
	arn := fmt.Sprintf("arn:aws:athena:%s:%s:workgroup/%s", region, accountID, convert.ToValue(wg.Name))

	var engineVersion, resultConfig interface{}
	var enforceConfig, publishMetrics, requesterPays *bool
	var bytesScannedCutoff *int64

	if wg.Configuration != nil {
		enforceConfig = wg.Configuration.EnforceWorkGroupConfiguration
		publishMetrics = wg.Configuration.PublishCloudWatchMetricsEnabled
		requesterPays = wg.Configuration.RequesterPaysEnabled
		bytesScannedCutoff = wg.Configuration.BytesScannedCutoffPerQuery
		var err error
		engineVersion, err = convert.JsonToDict(wg.Configuration.EngineVersion)
		if err != nil {
			return nil, err
		}
		resultConfig, err = convert.JsonToDict(wg.Configuration.ResultConfiguration)
		if err != nil {
			return nil, err
		}
	}

	resource, err := CreateResource(runtime, "aws.athena.workgroup",
		map[string]*llx.RawData{
			"__id":                            llx.StringData(arn),
			"arn":                             llx.StringData(arn),
			"name":                            llx.StringDataPtr(wg.Name),
			"state":                           llx.StringData(string(wg.State)),
			"description":                     llx.StringDataPtr(wg.Description),
			"createdAt":                       llx.TimeDataPtr(wg.CreationTime),
			"enforceWorkGroupConfiguration":   llx.BoolDataPtr(enforceConfig),
			"publishCloudWatchMetricsEnabled": llx.BoolDataPtr(publishMetrics),
			"bytesScannedCutoffPerQuery":      llx.IntDataPtr(bytesScannedCutoff),
			"requesterPaysEnabled":            llx.BoolDataPtr(requesterPays),
			"engineVersion":                   llx.DictData(engineVersion),
			"resultConfiguration":             llx.DictData(resultConfig),
			"region":                          llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsAthenaWorkgroup), nil
}

func (a *mqlAwsAthenaWorkgroup) tags() (map[string]interface{}, error) {
	if a.Arn.Error != nil {
		return nil, a.Arn.Error
	}
	if a.Region.Error != nil {
		return nil, a.Region.Error
	}
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Athena(region)
	ctx := context.Background()

	tags := make(map[string]interface{})
	var nextToken *string
	for {
		resp, err := svc.ListTagsForResource(ctx, &athena.ListTagsForResourceInput{
			ResourceARN: &arn,
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, tag := range resp.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return tags, nil
}
