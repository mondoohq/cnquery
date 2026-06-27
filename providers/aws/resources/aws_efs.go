// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efstypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/smithy-go/transport/http"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsEfsFilesystem) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEfs) filesystems() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFilesystems(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsEfs) getFilesystems(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("efs>getFilesystems>calling aws with region %s", region)

			svc := conn.Efs(region)
			ctx := context.Background()
			res := []any{}

			params := &efs.DescribeFileSystemsInput{}
			paginator := efs.NewDescribeFileSystemsPaginator(svc, params)
			for paginator.HasMorePages() {
				describeFileSystemsRes, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, fs := range describeFileSystemsRes.FileSystems {
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(efsTagsToMap(fs.Tags))) {
						log.Debug().Interface("filesystem", fs.FileSystemArn).Msg("skipping efs filesystem due to filters")
						continue
					}

					mqlFilesystem, err := buildEfsFilesystemResource(a.MqlRuntime, region, fs)
					if err != nil {
						return nil, err
					}

					res = append(res, mqlFilesystem)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsEfsFilesystemInternal struct {
	cacheKmsKeyID             *string
	cacheFileSystemProtection *efstypes.FileSystemProtectionDescription
}

func buildEfsFilesystemResource(runtime *plugin.Runtime, region string, fs efstypes.FileSystemDescription) (*mqlAwsEfsFilesystem, error) {
	var sizeInBytes int64
	if fs.SizeInBytes != nil {
		sizeInBytes = fs.SizeInBytes.Value
	}

	args := map[string]*llx.RawData{
		"id":               llx.StringDataPtr(fs.FileSystemId),
		"arn":              llx.StringDataPtr(fs.FileSystemArn),
		"name":             llx.StringDataPtr(fs.Name),
		"encrypted":        llx.BoolData(convert.ToValue(fs.Encrypted)),
		"ownerId":          llx.StringDataPtr(fs.OwnerId),
		"region":           llx.StringData(region),
		"availabilityZone": llx.StringDataPtr(fs.AvailabilityZoneName),
		"createdAt":        llx.TimeDataPtr(fs.CreationTime),
		"tags":             llx.MapData(efsTagsToMap(fs.Tags), types.String),
		"performanceMode":  llx.StringData(string(fs.PerformanceMode)),
		"throughputMode":   llx.StringData(string(fs.ThroughputMode)),
		"sizeInBytes":      llx.IntData(sizeInBytes),
		"lifecycleState":   llx.StringData(string(fs.LifeCycleState)),
	}
	mqlFilesystem, err := CreateResource(runtime, "aws.efs.filesystem", args)
	if err != nil {
		return nil, err
	}
	fsResource := mqlFilesystem.(*mqlAwsEfsFilesystem)
	fsResource.cacheKmsKeyID = fs.KmsKeyId
	fsResource.cacheFileSystemProtection = fs.FileSystemProtection
	return fsResource, nil
}

func (a *mqlAwsEfsFilesystem) kmsKey() (*mqlAwsKmsKey, error) {
	// add kms key if there is one
	if a.cacheKmsKeyID != nil {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyID),
		})
		return mqlKeyResource.(*mqlAwsKmsKey), err
	}
	a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull

	return nil, nil
}

func initAwsEfsFilesystem(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch efs filesystem")
	}

	arnVal := args["arn"].Value.(string)

	// Derive region + filesystem id for a single targeted DescribeFileSystems
	// call instead of listing every filesystem in every region.
	var region, fsId string
	if parsed, err := arn.Parse(arnVal); err == nil && strings.HasPrefix(parsed.Resource, "file-system/") {
		region = parsed.Region
		fsId = strings.TrimPrefix(parsed.Resource, "file-system/")
	}

	if region != "" && fsId != "" {
		conn := runtime.Connection.(*connection.AwsConnection)
		svc := conn.Efs(region)
		resp, err := svc.DescribeFileSystems(context.Background(), &efs.DescribeFileSystemsInput{
			FileSystemId: &fsId,
		})
		if err != nil {
			return nil, nil, err
		}
		if len(resp.FileSystems) > 0 {
			fs, err := buildEfsFilesystemResource(runtime, region, resp.FileSystems[0])
			if err != nil {
				return nil, nil, err
			}
			return args, fs, nil
		}
		return nil, nil, errors.New("efs filesystem does not exist")
	}

	// Fallback: scan the cached list when the ARN can't be parsed for a
	// targeted lookup.
	obj, err := CreateResource(runtime, "aws.efs", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	efs := obj.(*mqlAwsEfs)
	rawResources := efs.GetFilesystems()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	for _, rawResource := range rawResources.Data {
		fs := rawResource.(*mqlAwsEfsFilesystem)
		if fs.Arn.Data == arnVal {
			return args, fs, nil
		}
	}
	return nil, nil, errors.New("efs filesystem does not exist")
}

func (a *mqlAwsEfsFilesystem) backupPolicy() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	backupPolicy, err := svc.DescribeBackupPolicy(ctx, &efs.DescribeBackupPolicyInput{
		FileSystemId: &id,
	})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	res, err := convert.JsonToDict(backupPolicy)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsEfsFilesystem) lifecycleConfiguration() (*mqlAwsEfsFilesystemLifecycleConfiguration, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	resp, err := svc.DescribeLifecycleConfiguration(ctx, &efs.DescribeLifecycleConfigurationInput{
		FileSystemId: &id,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.LifecycleConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		var respErr *http.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			a.LifecycleConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}
	if len(resp.LifecyclePolicies) == 0 {
		a.LifecycleConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	// Aggregate all policies into a single lifecycle configuration
	var transitionToIA, transitionToArchive, transitionToPrimary string
	for _, p := range resp.LifecyclePolicies {
		if p.TransitionToIA != "" {
			transitionToIA = string(p.TransitionToIA)
		}
		if p.TransitionToArchive != "" {
			transitionToArchive = string(p.TransitionToArchive)
		}
		if p.TransitionToPrimaryStorageClass != "" {
			transitionToPrimary = string(p.TransitionToPrimaryStorageClass)
		}
	}

	configId := a.Arn.Data + "/lifecycleConfiguration"
	mqlConfig, err := CreateResource(a.MqlRuntime, "aws.efs.filesystem.lifecycleConfiguration",
		map[string]*llx.RawData{
			"__id":                            llx.StringData(configId),
			"transitionToIA":                  llx.StringData(transitionToIA),
			"transitionToArchive":             llx.StringData(transitionToArchive),
			"transitionToPrimaryStorageClass": llx.StringData(transitionToPrimary),
		})
	if err != nil {
		return nil, err
	}
	return mqlConfig.(*mqlAwsEfsFilesystemLifecycleConfiguration), nil
}

func (a *mqlAwsEfsFilesystemLifecycleConfiguration) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEfsFilesystem) replicationConfiguration() (*mqlAwsEfsFilesystemReplicationConfiguration, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	resp, err := svc.DescribeReplicationConfigurations(ctx, &efs.DescribeReplicationConfigurationsInput{
		FileSystemId: &id,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.ReplicationConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		var respErr *http.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			a.ReplicationConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}
	if len(resp.Replications) == 0 {
		a.ReplicationConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	repl := resp.Replications[0]

	// Build destination sub-resources
	destinations := make([]any, 0, len(repl.Destinations))
	for _, dest := range repl.Destinations {
		destId := convert.ToValue(repl.SourceFileSystemArn) + "/replication/" + convert.ToValue(dest.FileSystemId)
		mqlDest, err := CreateResource(a.MqlRuntime, "aws.efs.filesystem.replicationDestination",
			map[string]*llx.RawData{
				"__id":                    llx.StringData(destId),
				"fileSystemId":            llx.StringDataPtr(dest.FileSystemId),
				"region":                  llx.StringDataPtr(dest.Region),
				"status":                  llx.StringData(string(dest.Status)),
				"lastReplicatedTimestamp": llx.TimeDataPtr(dest.LastReplicatedTimestamp),
			})
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, mqlDest)
	}

	replId := convert.ToValue(repl.SourceFileSystemArn) + "/replication"
	mqlRepl, err := CreateResource(a.MqlRuntime, "aws.efs.filesystem.replicationConfiguration",
		map[string]*llx.RawData{
			"__id":                        llx.StringData(replId),
			"sourceFileSystemId":          llx.StringDataPtr(repl.SourceFileSystemId),
			"sourceFileSystemRegion":      llx.StringDataPtr(repl.SourceFileSystemRegion),
			"sourceFileSystemArn":         llx.StringDataPtr(repl.SourceFileSystemArn),
			"originalSourceFileSystemArn": llx.StringDataPtr(repl.OriginalSourceFileSystemArn),
			"creationTime":                llx.TimeDataPtr(repl.CreationTime),
			"destinations":                llx.ArrayData(destinations, types.Resource("aws.efs.filesystem.replicationDestination")),
		})
	if err != nil {
		return nil, err
	}
	return mqlRepl.(*mqlAwsEfsFilesystemReplicationConfiguration), nil
}

func (a *mqlAwsEfsFilesystemReplicationConfiguration) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEfsFilesystemReplicationDestination) id() (string, error) {
	return a.__id, nil
}

func (a *mqlAwsEfsFilesystem) fileSystemProtection() (any, error) {
	if a.cacheFileSystemProtection == nil {
		return nil, nil
	}
	result := map[string]any{
		"replicationOverwriteProtection": string(a.cacheFileSystemProtection.ReplicationOverwriteProtection),
	}
	return result, nil
}

func efsTagsToMap(tags []efstypes.Tag) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}

func (a *mqlAwsEfsFilesystem) mountTargets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	res := []any{}
	params := &efs.DescribeMountTargetsInput{
		FileSystemId: &id,
	}
	paginator := efs.NewDescribeMountTargetsPaginator(svc, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Str("fileSystemId", id).Msg("error accessing EFS mount targets")
				return res, nil
			}
			return nil, err
		}

		for _, mt := range page.MountTargets {
			// Fetch security groups for this mount target
			sgRes, err := svc.DescribeMountTargetSecurityGroups(ctx, &efs.DescribeMountTargetSecurityGroupsInput{
				MountTargetId: mt.MountTargetId,
			})
			if err != nil {
				log.Warn().Str("mountTargetId", convert.ToValue(mt.MountTargetId)).Msg("error fetching security groups for mount target")
			}

			args := map[string]*llx.RawData{
				"__id":               llx.StringDataPtr(mt.MountTargetId),
				"mountTargetId":      llx.StringDataPtr(mt.MountTargetId),
				"fileSystemId":       llx.StringDataPtr(mt.FileSystemId),
				"subnetId":           llx.StringDataPtr(mt.SubnetId),
				"availabilityZone":   llx.StringDataPtr(mt.AvailabilityZoneName),
				"ipAddress":          llx.StringDataPtr(mt.IpAddress),
				"lifecycleState":     llx.StringData(string(mt.LifeCycleState)),
				"networkInterfaceId": llx.StringDataPtr(mt.NetworkInterfaceId),
				"region":             llx.StringData(region),
			}

			mqlMountTarget, err := CreateResource(a.MqlRuntime, ResourceAwsEfsMountTarget, args)
			if err != nil {
				return nil, err
			}

			// Cache the security group IDs for lazy loading
			if sgRes != nil && len(sgRes.SecurityGroups) > 0 {
				mqlMountTarget.(*mqlAwsEfsMountTarget).cacheSecurityGroupIDs = sgRes.SecurityGroups
			}

			res = append(res, mqlMountTarget)
		}
	}

	return res, nil
}

func (a *mqlAwsEfsFilesystem) accessPoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	res := []any{}
	params := &efs.DescribeAccessPointsInput{
		FileSystemId: &id,
	}
	paginator := efs.NewDescribeAccessPointsPaginator(svc, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Str("fileSystemId", id).Msg("error accessing EFS access points")
				return res, nil
			}
			return nil, err
		}

		for _, ap := range page.AccessPoints {
			// Convert POSIX user to dict
			var posixUser map[string]any
			if ap.PosixUser != nil {
				posixUser = map[string]any{
					"uid": convert.ToValue(ap.PosixUser.Uid),
					"gid": convert.ToValue(ap.PosixUser.Gid),
				}
				if len(ap.PosixUser.SecondaryGids) > 0 {
					secondaryGids := make([]any, len(ap.PosixUser.SecondaryGids))
					for i, gid := range ap.PosixUser.SecondaryGids {
						secondaryGids[i] = gid
					}
					posixUser["secondaryGids"] = secondaryGids
				}
			}

			// Convert root directory to dict
			var rootDirectory map[string]any
			if ap.RootDirectory != nil {
				rootDirectory = map[string]any{
					"path": convert.ToValue(ap.RootDirectory.Path),
				}
				if ap.RootDirectory.CreationInfo != nil {
					rootDirectory["creationInfo"] = map[string]any{
						"ownerUid":    convert.ToValue(ap.RootDirectory.CreationInfo.OwnerUid),
						"ownerGid":    convert.ToValue(ap.RootDirectory.CreationInfo.OwnerGid),
						"permissions": convert.ToValue(ap.RootDirectory.CreationInfo.Permissions),
					}
				}
			}

			args := map[string]*llx.RawData{
				"__id":           llx.StringDataPtr(ap.AccessPointArn),
				"accessPointId":  llx.StringDataPtr(ap.AccessPointId),
				"arn":            llx.StringDataPtr(ap.AccessPointArn),
				"fileSystemId":   llx.StringDataPtr(ap.FileSystemId),
				"name":           llx.StringDataPtr(ap.Name),
				"lifecycleState": llx.StringData(string(ap.LifeCycleState)),
				"region":         llx.StringData(region),
				"tags":           llx.MapData(efsTagsToMap(ap.Tags), types.String),
			}

			if posixUser != nil {
				args["posixUser"] = llx.DictData(posixUser)
			}
			if rootDirectory != nil {
				args["rootDirectory"] = llx.DictData(rootDirectory)
			}

			mqlAccessPoint, err := CreateResource(a.MqlRuntime, ResourceAwsEfsAccessPoint, args)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAccessPoint)
		}
	}

	return res, nil
}

func (a *mqlAwsEfsFilesystem) fileSystemPolicy() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	id := a.Id.Data
	region := a.Region.Data

	svc := conn.Efs(region)
	ctx := context.Background()

	policyRes, err := svc.DescribeFileSystemPolicy(ctx, &efs.DescribeFileSystemPolicyInput{
		FileSystemId: &id,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("region", region).Str("fileSystemId", id).Msg("error accessing EFS file system policy")
			return "", nil
		}
		var respErr *http.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			// No policy exists
			return "", nil
		}
		return "", err
	}

	if policyRes != nil && policyRes.Policy != nil {
		return *policyRes.Policy, nil
	}

	return "", nil
}

// Mount Target implementation
type mqlAwsEfsMountTargetInternal struct {
	cacheSecurityGroupIDs []string
}

func (a *mqlAwsEfsMountTarget) securityGroups() ([]any, error) {
	if len(a.cacheSecurityGroupIDs) == 0 {
		return []any{}, nil
	}

	region := a.Region.Data

	res := []any{}
	for _, sgID := range a.cacheSecurityGroupIDs {
		mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup", map[string]*llx.RawData{
			"id":     llx.StringData(sgID),
			"region": llx.StringData(region),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSg)
	}

	return res, nil
}

func (a *mqlAwsEfsMountTarget) subnet() (*mqlAwsVpcSubnet, error) {
	subnetId := a.SubnetId.Data
	if subnetId == "" {
		a.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	subnetArn := fmt.Sprintf(subnetArnPattern, a.Region.Data, conn.AccountId(), subnetId)
	res, err := NewResource(a.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(subnetArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpcSubnet), nil
}
