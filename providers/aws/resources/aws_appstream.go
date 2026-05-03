// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
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
			"disableIMDSV1":                  llx.BoolDataPtr(fleet.DisableIMDSV1),
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

	stackNamesLock    sync.Mutex
	stackNamesFetched bool
	stackNames        []string
}

func (a *mqlAwsAppstreamFleet) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsAppstreamFleet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	region, name, err := parseAppstreamRef(args, "fleet/")
	if err != nil {
		return nil, nil, err
	}

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(region)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := svc.DescribeFleets(ctx, &appstream.DescribeFleetsInput{
		Names: []string{name},
	})
	if err != nil {
		if isAppstreamRegionError(err) {
			return args, nil, nil
		}
		return nil, nil, err
	}
	if len(resp.Fleets) == 0 {
		return args, nil, nil
	}

	mqlFleet, err := newMqlAwsAppstreamFleet(runtime, region, resp.Fleets[0])
	if err != nil {
		return nil, nil, err
	}
	return args, mqlFleet, nil
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

func initAwsAppstreamStack(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	region, name, err := parseAppstreamRef(args, "stack/")
	if err != nil {
		return nil, nil, err
	}

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(region)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := svc.DescribeStacks(ctx, &appstream.DescribeStacksInput{
		Names: []string{name},
	})
	if err != nil {
		if isAppstreamRegionError(err) {
			return args, nil, nil
		}
		return nil, nil, err
	}
	if len(resp.Stacks) == 0 {
		return args, nil, nil
	}

	mqlStack, err := newMqlAwsAppstreamStack(runtime, region, resp.Stacks[0])
	if err != nil {
		return nil, nil, err
	}
	return args, mqlStack, nil
}

// parseAppstreamRef extracts region + resource name from init args. Supports
// {arn}, {name + region}, or both. resourcePrefix is "fleet/" or "stack/".
func parseAppstreamRef(args map[string]*llx.RawData, resourcePrefix string) (string, string, error) {
	var region, name string
	if a := args["arn"]; a != nil {
		s, ok := a.Value.(string)
		if !ok {
			return "", "", errors.New("aws appstream init: arn must be a string")
		}
		parsed, err := arn.Parse(s)
		if err != nil {
			return "", "", err
		}
		region = parsed.Region
		name = strings.TrimPrefix(parsed.Resource, resourcePrefix)
	}
	if r := args["region"]; r != nil {
		if s, ok := r.Value.(string); ok {
			region = s
		}
	}
	if n := args["name"]; n != nil {
		if s, ok := n.Value.(string); ok {
			name = s
		}
	}
	if region == "" || name == "" {
		return "", "", errors.New("arn or (name + region) required to fetch aws appstream resource")
	}
	return region, name, nil
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
			"disableIMDSV1":               llx.BoolDataPtr(ib.DisableIMDSV1),
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

// Applications

func (a *mqlAwsAppstream) applications() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	pool := jobpool.CreatePool(a.getApplications(conn), 5)
	pool.Run()
	if pool.HasErrors() {
		return nil, pool.GetErrors()
	}
	for i := range pool.Jobs {
		if pool.Jobs[i].Result != nil {
			res = append(res, pool.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsAppstream) getApplications(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getApplications>region %s", region)
			svc := conn.Appstream(region)
			res := []any{}
			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeApplications(ctx, &appstream.DescribeApplicationsInput{NextToken: nextToken})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream applications API")
						return res, nil
					}
					return nil, err
				}
				for _, app := range resp.Applications {
					mqlApp, err := newMqlAwsAppstreamApplication(a.MqlRuntime, region, app)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlApp)
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

func newMqlAwsAppstreamApplication(runtime *plugin.Runtime, region string, app appstreamtypes.Application) (*mqlAwsAppstreamApplication, error) {
	platforms := []any{}
	for _, p := range app.Platforms {
		platforms = append(platforms, string(p))
	}
	resource, err := CreateResource(runtime, "aws.appstream.application",
		map[string]*llx.RawData{
			"__id":             llx.StringDataPtr(app.Arn),
			"arn":              llx.StringDataPtr(app.Arn),
			"name":             llx.StringDataPtr(app.Name),
			"displayName":      llx.StringDataPtr(app.DisplayName),
			"description":      llx.StringDataPtr(app.Description),
			"enabled":          llx.BoolDataPtr(app.Enabled),
			"launchPath":       llx.StringDataPtr(app.LaunchPath),
			"launchParameters": llx.StringDataPtr(app.LaunchParameters),
			"workingDirectory": llx.StringDataPtr(app.WorkingDirectory),
			"platforms":        llx.ArrayData(platforms, types.String),
			"instanceFamilies": llx.ArrayData(toInterfaceArr(app.InstanceFamilies), types.String),
			"metadata":         llx.MapData(toInterfaceMap(app.Metadata), types.String),
			"appBlockArn":      llx.StringDataPtr(app.AppBlockArn),
			"createdAt":        llx.TimeDataPtr(app.CreatedTime),
			"region":           llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsAppstreamApplication), nil
}

func (a *mqlAwsAppstreamApplication) id() (string, error) { return a.Arn.Data, nil }

// Images

func (a *mqlAwsAppstream) images() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	pool := jobpool.CreatePool(a.getImages(conn), 5)
	pool.Run()
	if pool.HasErrors() {
		return nil, pool.GetErrors()
	}
	for i := range pool.Jobs {
		if pool.Jobs[i].Result != nil {
			res = append(res, pool.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsAppstream) getImages(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getImages>region %s", region)
			svc := conn.Appstream(region)
			res := []any{}
			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeImages(ctx, &appstream.DescribeImagesInput{NextToken: nextToken})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream images API")
						return res, nil
					}
					return nil, err
				}
				for _, img := range resp.Images {
					mqlImg, err := newMqlAwsAppstreamImage(a.MqlRuntime, region, img)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlImg)
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

func newMqlAwsAppstreamImage(runtime *plugin.Runtime, region string, img appstreamtypes.Image) (*mqlAwsAppstreamImage, error) {
	resource, err := CreateResource(runtime, "aws.appstream.image",
		map[string]*llx.RawData{
			"__id":                        llx.StringDataPtr(img.Arn),
			"arn":                         llx.StringDataPtr(img.Arn),
			"name":                        llx.StringDataPtr(img.Name),
			"displayName":                 llx.StringDataPtr(img.DisplayName),
			"description":                 llx.StringDataPtr(img.Description),
			"baseImageArn":                llx.StringDataPtr(img.BaseImageArn),
			"state":                       llx.StringData(string(img.State)),
			"visibility":                  llx.StringData(string(img.Visibility)),
			"platform":                    llx.StringData(string(img.Platform)),
			"imageBuilderName":            llx.StringDataPtr(img.ImageBuilderName),
			"imageBuilderSupported":       llx.BoolDataPtr(img.ImageBuilderSupported),
			"dynamicAppProvidersEnabled":  llx.BoolData(img.DynamicAppProvidersEnabled == appstreamtypes.DynamicAppProvidersEnabledEnabled),
			"appstreamAgentVersion":       llx.StringDataPtr(img.AppstreamAgentVersion),
			"createdAt":                   llx.TimeDataPtr(img.CreatedTime),
			"publicBaseImageReleasedDate": llx.TimeDataPtr(img.PublicBaseImageReleasedDate),
			"region":                      llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsAppstreamImage), nil
}

func (a *mqlAwsAppstreamImage) id() (string, error) { return a.Arn.Data, nil }

func (a *mqlAwsAppstreamImage) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := svc.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
		ResourceArn: aws.String(a.Arn.Data),
	})
	if err != nil {
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}

// Users (USERPOOL only)

func (a *mqlAwsAppstream) users() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	pool := jobpool.CreatePool(a.getUsers(conn), 5)
	pool.Run()
	if pool.HasErrors() {
		return nil, pool.GetErrors()
	}
	for i := range pool.Jobs {
		if pool.Jobs[i].Result != nil {
			res = append(res, pool.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsAppstream) getUsers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("appstream>getUsers>region %s", region)
			svc := conn.Appstream(region)
			res := []any{}
			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.DescribeUsers(ctx, &appstream.DescribeUsersInput{
					AuthenticationType: appstreamtypes.AuthenticationTypeUserpool,
					NextToken:          nextToken,
				})
				cancel()
				if err != nil {
					if isAppstreamRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS AppStream users API")
						return res, nil
					}
					return nil, err
				}
				for _, u := range resp.Users {
					resource, err := CreateResource(a.MqlRuntime, "aws.appstream.user",
						map[string]*llx.RawData{
							"__id":               llx.StringDataPtr(u.Arn),
							"arn":                llx.StringDataPtr(u.Arn),
							"userName":           llx.StringDataPtr(u.UserName),
							"authenticationType": llx.StringData(string(u.AuthenticationType)),
							"firstName":          llx.StringDataPtr(u.FirstName),
							"lastName":           llx.StringDataPtr(u.LastName),
							"status":             llx.StringDataPtr(u.Status),
							"enabled":            llx.BoolDataPtr(u.Enabled),
							"createdAt":          llx.TimeDataPtr(u.CreatedTime),
							"region":             llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, resource)
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

func (a *mqlAwsAppstreamUser) id() (string, error) { return a.Arn.Data, nil }

// Stack ↔ Fleet associations + Entitlements + Sessions

func (a *mqlAwsAppstreamStack) associatedFleets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	res := []any{}
	var nextToken *string
	stackName := a.Name.Data
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := svc.ListAssociatedFleets(ctx, &appstream.ListAssociatedFleetsInput{
			StackName: aws.String(stackName),
			NextToken: nextToken,
		})
		cancel()
		if err != nil {
			if isAppstreamRegionError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, fleetName := range resp.Names {
			fleetArn := buildAppstreamFleetArn(a.Region.Data, conn.AccountId(), fleetName)
			fleet, err := NewResource(a.MqlRuntime, "aws.appstream.fleet",
				map[string]*llx.RawData{"arn": llx.StringData(fleetArn)})
			if err != nil {
				return nil, err
			}
			res = append(res, fleet)
		}
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return res, nil
}

// listAssociatedStackNames returns (and caches) the names of stacks associated
// with this fleet. Both associatedStacks() and sessions() walk the same set of
// stacks, so we fetch ListAssociatedStacks once per fleet.
func (a *mqlAwsAppstreamFleet) listAssociatedStackNames() ([]string, error) {
	a.stackNamesLock.Lock()
	defer a.stackNamesLock.Unlock()
	if a.stackNamesFetched {
		return a.stackNames, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	names := []string{}
	var nextToken *string
	fleetName := a.Name.Data
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := svc.ListAssociatedStacks(ctx, &appstream.ListAssociatedStacksInput{
			FleetName: aws.String(fleetName),
			NextToken: nextToken,
		})
		cancel()
		if err != nil {
			if isAppstreamRegionError(err) {
				log.Debug().Str("region", a.Region.Data).Str("fleet", fleetName).Int("partial", len(names)).Msg("error accessing region for AWS AppStream associated stacks API; caching partial result")
				a.stackNamesFetched = true
				a.stackNames = names
				return names, nil
			}
			return nil, err
		}
		names = append(names, resp.Names...)
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	a.stackNamesFetched = true
	a.stackNames = names
	return names, nil
}

func (a *mqlAwsAppstreamFleet) associatedStacks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	names, err := a.listAssociatedStackNames()
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(names))
	for _, stackName := range names {
		stackArn := buildAppstreamStackArn(a.Region.Data, conn.AccountId(), stackName)
		stack, err := NewResource(a.MqlRuntime, "aws.appstream.stack",
			map[string]*llx.RawData{"arn": llx.StringData(stackArn)})
		if err != nil {
			return nil, err
		}
		res = append(res, stack)
	}
	return res, nil
}

func buildAppstreamFleetArn(region, account, name string) string {
	return "arn:aws:appstream:" + region + ":" + account + ":fleet/" + name
}

func buildAppstreamStackArn(region, account, name string) string {
	return "arn:aws:appstream:" + region + ":" + account + ":stack/" + name
}

func (a *mqlAwsAppstreamStack) entitlements() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	res := []any{}
	var nextToken *string
	stackName := a.Name.Data
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := svc.DescribeEntitlements(ctx, &appstream.DescribeEntitlementsInput{
			StackName: aws.String(stackName),
			NextToken: nextToken,
		})
		cancel()
		if err != nil {
			if isAppstreamRegionError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, e := range resp.Entitlements {
			attrs := map[string]any{}
			for _, ea := range e.Attributes {
				attrs[aws.ToString(ea.Name)] = aws.ToString(ea.Value)
			}
			ename := aws.ToString(e.Name)
			synthArn := "arn:aws:appstream:" + a.Region.Data + ":" + conn.AccountId() + ":entitlement/" + stackName + "/" + ename
			resource, err := CreateResource(a.MqlRuntime, "aws.appstream.entitlement",
				map[string]*llx.RawData{
					"__id":           llx.StringData(synthArn),
					"arn":            llx.StringData(synthArn),
					"name":           llx.StringData(ename),
					"stackName":      llx.StringData(stackName),
					"appVisibility":  llx.StringData(string(e.AppVisibility)),
					"attributes":     llx.MapData(attrs, types.String),
					"description":    llx.StringDataPtr(e.Description),
					"createdAt":      llx.TimeDataPtr(e.CreatedTime),
					"lastModifiedAt": llx.TimeDataPtr(e.LastModifiedTime),
					"region":         llx.StringData(a.Region.Data),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return res, nil
}

func (a *mqlAwsAppstreamEntitlement) id() (string, error) { return a.Arn.Data, nil }

func (a *mqlAwsAppstreamEntitlement) stack() (*mqlAwsAppstreamStack, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	stackArn := buildAppstreamStackArn(a.Region.Data, conn.AccountId(), a.StackName.Data)
	res, err := NewResource(a.MqlRuntime, "aws.appstream.stack",
		map[string]*llx.RawData{"arn": llx.StringData(stackArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsAppstreamStack), nil
}

// Sessions — fetched per-fleet across all stacks the fleet is associated with.

func (a *mqlAwsAppstreamFleet) sessions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Appstream(a.Region.Data)
	res := []any{}

	// Walk each associated stack, then list active sessions for the (fleet, stack) pair.
	// Stack names are cached on the fleet so associatedStacks() and sessions() share the same listing.
	stackNames, err := a.listAssociatedStackNames()
	if err != nil {
		return nil, err
	}
	fleetName := a.Name.Data
	for _, stackName := range stackNames {
		var sessionToken *string
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			sresp, err := svc.DescribeSessions(ctx, &appstream.DescribeSessionsInput{
				FleetName: aws.String(fleetName),
				StackName: aws.String(stackName),
				NextToken: sessionToken,
			})
			cancel()
			if err != nil {
				if isAppstreamRegionError(err) {
					break
				}
				return nil, err
			}
			for _, s := range sresp.Sessions {
				sid := aws.ToString(s.Id)
				synthArn := "arn:aws:appstream:" + a.Region.Data + ":" + conn.AccountId() + ":session/" + sid
				var eniId, eniIp string
				if s.NetworkAccessConfiguration != nil {
					eniId = aws.ToString(s.NetworkAccessConfiguration.EniId)
					eniIp = aws.ToString(s.NetworkAccessConfiguration.EniPrivateIpAddress)
				}
				resource, err := CreateResource(a.MqlRuntime, "aws.appstream.session",
					map[string]*llx.RawData{
						"__id":                llx.StringData(synthArn),
						"id":                  llx.StringData(sid),
						"userId":              llx.StringDataPtr(s.UserId),
						"state":               llx.StringData(string(s.State)),
						"connectionState":     llx.StringData(string(s.ConnectionState)),
						"authenticationType":  llx.StringData(string(s.AuthenticationType)),
						"fleetName":           llx.StringDataPtr(s.FleetName),
						"stackName":           llx.StringDataPtr(s.StackName),
						"instanceId":          llx.StringDataPtr(s.InstanceId),
						"startTime":           llx.TimeDataPtr(s.StartTime),
						"maxExpirationTime":   llx.TimeDataPtr(s.MaxExpirationTime),
						"eniId":               llx.StringData(eniId),
						"eniPrivateIpAddress": llx.StringData(eniIp),
						"region":              llx.StringData(a.Region.Data),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}
			if sresp.NextToken == nil {
				break
			}
			sessionToken = sresp.NextToken
		}
	}
	return res, nil
}

func (a *mqlAwsAppstreamSession) id() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return "arn:aws:appstream:" + a.Region.Data + ":" + conn.AccountId() + ":session/" + a.Id.Data, nil
}

func (a *mqlAwsAppstreamSession) fleet() (*mqlAwsAppstreamFleet, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	fleetArn := buildAppstreamFleetArn(a.Region.Data, conn.AccountId(), a.FleetName.Data)
	res, err := NewResource(a.MqlRuntime, "aws.appstream.fleet",
		map[string]*llx.RawData{"arn": llx.StringData(fleetArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsAppstreamFleet), nil
}

func (a *mqlAwsAppstreamSession) stack() (*mqlAwsAppstreamStack, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	stackArn := buildAppstreamStackArn(a.Region.Data, conn.AccountId(), a.StackName.Data)
	res, err := NewResource(a.MqlRuntime, "aws.appstream.stack",
		map[string]*llx.RawData{"arn": llx.StringData(stackArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsAppstreamStack), nil
}
