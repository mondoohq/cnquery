// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/directoryservice"
	dstypes "github.com/aws/aws-sdk-go-v2/service/directoryservice/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsDirectoryservice) id() (string, error) {
	return "aws.directoryservice", nil
}

func (a *mqlAwsDirectoryservice) directories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDirectories(conn), 5)
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

func (a *mqlAwsDirectoryservice) getDirectories(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("directoryservice>getDirectories>calling aws with region %s", region)
			svc := conn.DirectoryService(region)
			ctx := context.Background()
			res := []any{}

			paginator := directoryservice.NewDescribeDirectoriesPaginator(svc, &directoryservice.DescribeDirectoriesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS Directory Service API")
						return res, nil
					}
					return nil, err
				}

				for _, dir := range page.DirectoryDescriptions {
					mqlDir, err := newMqlAwsDirectoryserviceDirectory(a.MqlRuntime, region, dir)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDir)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsDirectoryserviceDirectory(runtime *plugin.Runtime, region string, dir dstypes.DirectoryDescription) (*mqlAwsDirectoryserviceDirectory, error) {
	ownerDesc, err := convert.JsonToDict(dir.OwnerDirectoryDescription)
	if err != nil {
		return nil, err
	}

	dirId := aws.ToString(dir.DirectoryId)

	resource, err := CreateResource(runtime, "aws.directoryservice.directory",
		map[string]*llx.RawData{
			"__id":                             llx.StringData("aws.directoryservice.directory/" + region + "/" + dirId),
			"directoryId":                      llx.StringData(dirId),
			"name":                             llx.StringDataPtr(dir.Name),
			"shortName":                        llx.StringDataPtr(dir.ShortName),
			"description":                      llx.StringDataPtr(dir.Description),
			"type":                             llx.StringData(string(dir.Type)),
			"edition":                          llx.StringData(string(dir.Edition)),
			"size":                             llx.StringData(string(dir.Size)),
			"alias":                            llx.StringDataPtr(dir.Alias),
			"accessUrl":                        llx.StringDataPtr(dir.AccessUrl),
			"stage":                            llx.StringData(string(dir.Stage)),
			"stageReason":                      llx.StringDataPtr(dir.StageReason),
			"stageLastUpdatedDateTime":         llx.TimeDataPtr(dir.StageLastUpdatedDateTime),
			"launchTime":                       llx.TimeDataPtr(dir.LaunchTime),
			"dnsIpAddrs":                       llx.ArrayData(toInterfaceArr(dir.DnsIpAddrs), types.String),
			"ssoEnabled":                       llx.BoolData(dir.SsoEnabled),
			"desiredNumberOfDomainControllers": llx.IntDataDefault(dir.DesiredNumberOfDomainControllers, 0),
			"osVersion":                        llx.StringData(string(dir.OsVersion)),
			"radiusStatus":                     llx.StringData(string(dir.RadiusStatus)),
			"ownerDirectoryDescription":        llx.MapData(ownerDesc, types.Any),
			"shareMethod":                      llx.StringData(string(dir.ShareMethod)),
			"shareStatus":                      llx.StringData(string(dir.ShareStatus)),
			"shareNotes":                       llx.StringDataPtr(dir.ShareNotes),
			"region":                           llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}

	mqlDir := resource.(*mqlAwsDirectoryserviceDirectory)
	mqlDir.dir = dir
	return mqlDir, nil
}

type mqlAwsDirectoryserviceDirectoryInternal struct {
	dir dstypes.DirectoryDescription
}

func (a *mqlAwsDirectoryserviceDirectory) id() (string, error) {
	return "aws.directoryservice.directory/" + a.Region.Data + "/" + a.DirectoryId.Data, nil
}

func (a *mqlAwsDirectoryserviceDirectory) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.DirectoryService(a.Region.Data)
	ctx := context.Background()

	tagsMap := make(map[string]any)
	paginator := directoryservice.NewListTagsForResourcePaginator(svc, &directoryservice.ListTagsForResourceInput{
		ResourceId: aws.String(a.DirectoryId.Data),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tag := range page.Tags {
			tagsMap[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}
	return tagsMap, nil
}

func (a *mqlAwsDirectoryserviceDirectory) radiusSettings() (*mqlAwsDirectoryserviceRadiusSettings, error) {
	if a.dir.RadiusSettings == nil {
		a.RadiusSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	rs := a.dir.RadiusSettings
	dirId := a.DirectoryId.Data
	region := a.Region.Data

	resource, err := CreateResource(a.MqlRuntime, "aws.directoryservice.radiusSettings",
		map[string]*llx.RawData{
			"__id":                   llx.StringData("aws.directoryservice.radiusSettings/" + region + "/" + dirId),
			"authenticationProtocol": llx.StringData(string(rs.AuthenticationProtocol)),
			"displayLabel":           llx.StringDataPtr(rs.DisplayLabel),
			"radiusPort":             llx.IntDataDefault(rs.RadiusPort, 0),
			"radiusRetries":          llx.IntData(rs.RadiusRetries),
			"radiusServers":          llx.ArrayData(toInterfaceArr(rs.RadiusServers), types.String),
			"radiusTimeout":          llx.IntDataDefault(rs.RadiusTimeout, 0),
			"useSameUsername":        llx.BoolData(rs.UseSameUsername),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsDirectoryserviceRadiusSettings), nil
}

func (a *mqlAwsDirectoryserviceRadiusSettings) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsDirectoryserviceDirectory) vpcSettings() (*mqlAwsDirectoryserviceVpcSettings, error) {
	if a.dir.VpcSettings == nil {
		a.VpcSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	vs := a.dir.VpcSettings
	dirId := a.DirectoryId.Data
	region := a.Region.Data

	resource, err := CreateResource(a.MqlRuntime, "aws.directoryservice.vpcSettings",
		map[string]*llx.RawData{
			"__id":              llx.StringData("aws.directoryservice.vpcSettings/" + region + "/" + dirId),
			"vpcId":             llx.StringDataPtr(vs.VpcId),
			"securityGroupId":   llx.StringDataPtr(vs.SecurityGroupId),
			"subnetIds":         llx.ArrayData(toInterfaceArr(vs.SubnetIds), types.String),
			"availabilityZones": llx.ArrayData(toInterfaceArr(vs.AvailabilityZones), types.String),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsDirectoryserviceVpcSettings), nil
}

func (a *mqlAwsDirectoryserviceVpcSettings) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsDirectoryserviceDirectory) connectSettings() (*mqlAwsDirectoryserviceConnectSettings, error) {
	if a.dir.ConnectSettings == nil {
		a.ConnectSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	cs := a.dir.ConnectSettings
	dirId := a.DirectoryId.Data
	region := a.Region.Data

	resource, err := CreateResource(a.MqlRuntime, "aws.directoryservice.connectSettings",
		map[string]*llx.RawData{
			"__id":              llx.StringData("aws.directoryservice.connectSettings/" + region + "/" + dirId),
			"vpcId":             llx.StringDataPtr(cs.VpcId),
			"securityGroupId":   llx.StringDataPtr(cs.SecurityGroupId),
			"subnetIds":         llx.ArrayData(toInterfaceArr(cs.SubnetIds), types.String),
			"availabilityZones": llx.ArrayData(toInterfaceArr(cs.AvailabilityZones), types.String),
			"connectIps":        llx.ArrayData(toInterfaceArr(cs.ConnectIps), types.String),
			"customerUserName":  llx.StringDataPtr(cs.CustomerUserName),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsDirectoryserviceConnectSettings), nil
}

func (a *mqlAwsDirectoryserviceConnectSettings) id() (string, error) {
	return a.__id, nil
}
