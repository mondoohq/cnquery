// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appstream"
	appstreamtypes "github.com/aws/aws-sdk-go-v2/service/appstream/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

// isAppstreamRegionError checks if the error indicates the AppStream service
// is not available or reachable in the given region. This includes DNS failures
// (covered by IsServiceNotAvailableInRegionError) and context deadline exceeded
// errors from the per-request timeout used to avoid long waits on regions where
// the endpoint exists but the service is unavailable.
func isAppstreamRegionError(err error) bool {
	return Is400AccessDeniedError(err) ||
		IsServiceNotAvailableInRegionError(err) ||
		errors.Is(err, context.DeadlineExceeded)
}

func (a *mqlAwsAppstream) id() (string, error) {
	return "aws.appstream", nil
}

// Fleets

func (a *mqlAwsAppstream) fleets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFleets(conn), 5)
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

func (a *mqlAwsAppstream) getFleets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getFleets>calling aws with region %s", region)
			svc := conn.Appstream(region)
			res := []any{}

			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeFleets(ctx, &appstream.DescribeFleetsInput{
					NextToken: nextToken,
				})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream fleet API")
						return res, nil
					}
					return nil, err
				}

				for _, fleet := range resp.Fleets {
					mqlFleet, err := newMqlAwsAppstreamFleet(a.MqlRuntime, region, fleet)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFleet)
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

func newMqlAwsAppstreamFleet(runtime *plugin.Runtime, region string, fleet appstreamtypes.Fleet) (*mqlAwsAppstreamFleet, error) {
	domainJoinInfo, err := convert.JsonToDict(fleet.DomainJoinInfo)
	if err != nil {
		return nil, err
	}
	vpcConfig, err := convert.JsonToDict(fleet.VpcConfig)
	if err != nil {
		return nil, err
	}
	resource, err := CreateResource(runtime, "aws.appstream.fleet",
		map[string]*llx.RawData{
			"__id":                           llx.StringDataPtr(fleet.Arn),
			"arn":                            llx.StringDataPtr(fleet.Arn),
			"name":                           llx.StringDataPtr(fleet.Name),
			"state":                          llx.StringData(string(fleet.State)),
			"fleetType":                      llx.StringData(string(fleet.FleetType)),
			"instanceType":                   llx.StringDataPtr(fleet.InstanceType),
			"maxUserDurationInSeconds":       llx.IntDataDefault(fleet.MaxUserDurationInSeconds, 0),
			"disconnectTimeoutInSeconds":     llx.IntDataDefault(fleet.DisconnectTimeoutInSeconds, 0),
			"idleDisconnectTimeoutInSeconds": llx.IntDataDefault(fleet.IdleDisconnectTimeoutInSeconds, 0),
			"enableDefaultInternetAccess":    llx.BoolDataPtr(fleet.EnableDefaultInternetAccess),
			"domainJoinInfo":                 llx.MapData(domainJoinInfo, types.Any),
			"maxConcurrentSessions":          llx.IntDataDefault(fleet.MaxConcurrentSessions, 0),
			"maxSessionsPerInstance":         llx.IntDataDefault(fleet.MaxSessionsPerInstance, 0),
			"vpcConfig":                      llx.MapData(vpcConfig, types.Any),
			"iamRoleArn":                     llx.StringDataPtr(fleet.IamRoleArn),
			"imageName":                      llx.StringDataPtr(fleet.ImageName),
			"imageArn":                       llx.StringDataPtr(fleet.ImageArn),
			"platform":                       llx.StringData(string(fleet.Platform)),
			"createdAt":                      llx.TimeDataPtr(fleet.CreatedTime),
			"region":                         llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlFleet := resource.(*mqlAwsAppstreamFleet)
	mqlFleet.cacheComputeCapacityStatus = fleet.ComputeCapacityStatus
	return mqlFleet, nil
}

type mqlAwsAppstreamFleetInternal struct {
	cacheComputeCapacityStatus *appstreamtypes.ComputeCapacityStatus
}

func (a *mqlAwsAppstreamFleet) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAppstreamFleet) iamRole() (*mqlAwsIamRole, error) {
	arnVal := a.IamRoleArn.Data
	if arnVal == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsAppstreamFleet) computeCapacityStatus() (*mqlAwsAppstreamFleetComputeCapacityStatus, error) {
	ccs := a.cacheComputeCapacityStatus
	if ccs == nil {
		a.ComputeCapacityStatus.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "aws.appstream.fleet.computeCapacityStatus",
		map[string]*llx.RawData{
			"__id":                        llx.StringData(a.Arn.Data + "/computeCapacityStatus"),
			"desired":                     llx.IntDataDefault(ccs.Desired, 0),
			"running":                     llx.IntDataDefault(ccs.Running, 0),
			"inUse":                       llx.IntDataDefault(ccs.InUse, 0),
			"available":                   llx.IntDataDefault(ccs.Available, 0),
			"activeUserSessions":          llx.IntDataDefault(ccs.ActiveUserSessions, 0),
			"actualUserSessions":          llx.IntDataDefault(ccs.ActualUserSessions, 0),
			"availableUserSessions":       llx.IntDataDefault(ccs.AvailableUserSessions, 0),
			"desiredUserSessions":         llx.IntDataDefault(ccs.DesiredUserSessions, 0),
			"draining":                    llx.IntDataDefault(ccs.Draining, 0),
			"drainModeActiveUserSessions": llx.IntDataDefault(ccs.DrainModeActiveUserSessions, 0),
			"drainModeUnusedUserSessions": llx.IntDataDefault(ccs.DrainModeUnusedUserSessions, 0),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsAppstreamFleetComputeCapacityStatus), nil
}

func (a *mqlAwsAppstreamFleetComputeCapacityStatus) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsAppstreamFleet) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
		ResourceArn: aws.String(a.Arn.Data),
	})
	if err != nil {
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}

// Stacks

func (a *mqlAwsAppstream) stacks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStacks(conn), 5)
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

func (a *mqlAwsAppstream) getStacks(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getStacks>calling aws with region %s", region)
			svc := conn.Appstream(region)
			res := []any{}

			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeStacks(ctx, &appstream.DescribeStacksInput{
					NextToken: nextToken,
				})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream stack API")
						return res, nil
					}
					return nil, err
				}

				for _, stack := range resp.Stacks {
					mqlStack, err := newMqlAwsAppstreamStack(a.MqlRuntime, region, stack)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlStack)
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

func newMqlAwsAppstreamStack(runtime *plugin.Runtime, region string, stack appstreamtypes.Stack) (*mqlAwsAppstreamStack, error) {
	accessEndpoints, err := convert.JsonToDictSlice(stack.AccessEndpoints)
	if err != nil {
		return nil, err
	}
	applicationSettings, err := convert.JsonToDict(stack.ApplicationSettings)
	if err != nil {
		return nil, err
	}
	storageConnectors, err := convert.JsonToDictSlice(stack.StorageConnectors)
	if err != nil {
		return nil, err
	}
	userSettings, err := convert.JsonToDictSlice(stack.UserSettings)
	if err != nil {
		return nil, err
	}
	resource, err := CreateResource(runtime, "aws.appstream.stack",
		map[string]*llx.RawData{
			"__id":                llx.StringDataPtr(stack.Arn),
			"arn":                 llx.StringDataPtr(stack.Arn),
			"name":                llx.StringDataPtr(stack.Name),
			"description":         llx.StringDataPtr(stack.Description),
			"createdAt":           llx.TimeDataPtr(stack.CreatedTime),
			"accessEndpoints":     llx.ArrayData(accessEndpoints, types.Dict),
			"applicationSettings": llx.MapData(applicationSettings, types.Any),
			"storageConnectors":   llx.ArrayData(storageConnectors, types.Dict),
			"userSettings":        llx.ArrayData(userSettings, types.Dict),
			"embedHostDomains":    llx.ArrayData(toInterfaceArr(stack.EmbedHostDomains), types.String),
			"region":              llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlStack := resource.(*mqlAwsAppstreamStack)
	mqlStack.cacheContentRedirection = stack.ContentRedirection
	return mqlStack, nil
}

type mqlAwsAppstreamStackInternal struct {
	cacheContentRedirection *appstreamtypes.ContentRedirection
}

func (a *mqlAwsAppstreamStack) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAppstreamStack) contentRedirection() (*mqlAwsAppstreamStackContentRedirection, error) {
	cr := a.cacheContentRedirection
	if cr == nil {
		a.ContentRedirection.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var enabled bool
	var allowedUrls []string
	var deniedUrls []string
	if cr.HostToClient != nil {
		enabled = aws.ToBool(cr.HostToClient.Enabled)
		allowedUrls = cr.HostToClient.AllowedUrls
		deniedUrls = cr.HostToClient.DeniedUrls
	}

	res, err := CreateResource(a.MqlRuntime, "aws.appstream.stack.contentRedirection",
		map[string]*llx.RawData{
			"__id":                    llx.StringData(a.Arn.Data + "/contentRedirection"),
			"hostToClientEnabled":     llx.BoolData(enabled),
			"hostToClientAllowedUrls": llx.ArrayData(toInterfaceArr(allowedUrls), types.String),
			"hostToClientDeniedUrls":  llx.ArrayData(toInterfaceArr(deniedUrls), types.String),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsAppstreamStackContentRedirection), nil
}

func (a *mqlAwsAppstreamStackContentRedirection) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsAppstreamStack) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
		ResourceArn: aws.String(a.Arn.Data),
	})
	if err != nil {
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}

// Image Builders

func (a *mqlAwsAppstream) imageBuilders() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getImageBuilders(conn), 5)
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

func (a *mqlAwsAppstream) getImageBuilders(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getImageBuilders>calling aws with region %s", region)
			svc := conn.Appstream(region)
			res := []any{}

			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeImageBuilders(ctx, &appstream.DescribeImageBuildersInput{
					NextToken: nextToken,
				})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream image builder API")
						return res, nil
					}
					return nil, err
				}

				for _, ib := range resp.ImageBuilders {
					mqlIB, err := newMqlAwsAppstreamImageBuilder(a.MqlRuntime, region, ib)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlIB)
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

func newMqlAwsAppstreamImageBuilder(runtime *plugin.Runtime, region string, ib appstreamtypes.ImageBuilder) (*mqlAwsAppstreamImageBuilder, error) {
	domainJoinInfo, err := convert.JsonToDict(ib.DomainJoinInfo)
	if err != nil {
		return nil, err
	}
	vpcConfig, err := convert.JsonToDict(ib.VpcConfig)
	if err != nil {
		return nil, err
	}
	networkAccessConfig, err := convert.JsonToDict(ib.NetworkAccessConfiguration)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, "aws.appstream.imageBuilder",
		map[string]*llx.RawData{
			"__id":                        llx.StringDataPtr(ib.Arn),
			"arn":                         llx.StringDataPtr(ib.Arn),
			"name":                        llx.StringDataPtr(ib.Name),
			"description":                 llx.StringDataPtr(ib.Description),
			"state":                       llx.StringData(string(ib.State)),
			"instanceType":                llx.StringDataPtr(ib.InstanceType),
			"platform":                    llx.StringData(string(ib.Platform)),
			"imageArn":                    llx.StringDataPtr(ib.ImageArn),
			"appstreamAgentVersion":       llx.StringDataPtr(ib.AppstreamAgentVersion),
			"enableDefaultInternetAccess": llx.BoolDataPtr(ib.EnableDefaultInternetAccess),
			"domainJoinInfo":              llx.MapData(domainJoinInfo, types.Any),
			"vpcConfig":                   llx.MapData(vpcConfig, types.Any),
			"iamRoleArn":                  llx.StringDataPtr(ib.IamRoleArn),
			"networkAccessConfiguration":  llx.MapData(networkAccessConfig, types.Any),
			"createdAt":                   llx.TimeDataPtr(ib.CreatedTime),
			"region":                      llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsAppstreamImageBuilder), nil
}

func (a *mqlAwsAppstreamImageBuilder) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAppstreamImageBuilder) iamRole() (*mqlAwsIamRole, error) {
	arnVal := a.IamRoleArn.Data
	if arnVal == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsAppstreamImageBuilder) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
		ResourceArn: aws.String(a.Arn.Data),
	})
	if err != nil {
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}
