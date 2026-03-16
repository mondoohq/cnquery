// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsEventbridge) pipes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPipes(conn), 5)
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

func (a *mqlAwsEventbridge) getPipes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eventbridge>getPipes>calling aws with region %s", region)

			svc := conn.Pipes(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListPipes(ctx, &pipes.ListPipesInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("pipes not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, pipe := range resp.Pipes {
					mqlPipe, err := CreateResource(a.MqlRuntime, "aws.eventbridge.pipe",
						map[string]*llx.RawData{
							"__id":         llx.StringDataPtr(pipe.Arn),
							"arn":          llx.StringDataPtr(pipe.Arn),
							"name":         llx.StringDataPtr(pipe.Name),
							"region":       llx.StringData(region),
							"source":       llx.StringDataPtr(pipe.Source),
							"target":       llx.StringDataPtr(pipe.Target),
							"enrichment":   llx.StringDataPtr(pipe.Enrichment),
							"currentState": llx.StringData(string(pipe.CurrentState)),
							"desiredState": llx.StringData(string(pipe.DesiredState)),
							"stateReason":  llx.StringDataPtr(pipe.StateReason),
							"createdAt":    llx.TimeDataPtr(pipe.CreationTime),
							"updatedAt":    llx.TimeDataPtr(pipe.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					mqlPipeRes := mqlPipe.(*mqlAwsEventbridgePipe)
					mqlPipeRes.cacheName = pipe.Name
					mqlPipeRes.cacheRegion = region
					res = append(res, mqlPipeRes)
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

type mqlAwsEventbridgePipeInternal struct {
	cacheName    *string
	cacheRegion  string
	cacheRoleArn *string
	fetched      bool
	lock         sync.Mutex
}

func (a *mqlAwsEventbridgePipe) fetchDetails() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}

	if a.cacheName == nil {
		a.fetched = true
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Pipes(a.cacheRegion)
	ctx := context.Background()

	resp, err := svc.DescribePipe(ctx, &pipes.DescribePipeInput{
		Name: a.cacheName,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("pipe", *a.cacheName).Msg("access denied describing pipe")
			a.fetched = true
			return nil
		}
		return err
	}

	a.cacheRoleArn = resp.RoleArn
	if resp.Description != nil {
		a.Description = plugin.TValue[string]{Data: *resp.Description, State: plugin.StateIsSet}
	} else {
		a.Description = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
	}
	tags := make(map[string]any)
	for k, v := range resp.Tags {
		tags[k] = v
	}
	a.Tags = plugin.TValue[map[string]any]{Data: tags, State: plugin.StateIsSet}

	a.fetched = true
	return nil
}

func (a *mqlAwsEventbridgePipe) description() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsEventbridgePipe) tags() (map[string]any, error) {
	return nil, a.fetchDetails()
}

func (a *mqlAwsEventbridgePipe) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
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
