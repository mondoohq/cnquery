// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/fsx"
	fsxtypes "github.com/aws/aws-sdk-go-v2/service/fsx/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

// ========================
// aws.fsx
// ========================

func (a *mqlAwsFsx) id() (string, error) {
	return ResourceAwsFsx, nil
}

// ========================
// aws.fsx.filesystem
// ========================

func (a *mqlAwsFsxFilesystem) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsFsx) fileSystems() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFileSystems(conn), 5)
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

func (a *mqlAwsFsx) getFileSystems(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("fsx>getFileSystems>calling aws with region %s", regionVal)

			svc := conn.Fsx(regionVal)
			ctx := context.Background()
			res := []any{}

			paginator := fsx.NewDescribeFileSystemsPaginator(svc, &fsx.DescribeFileSystemsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, fs := range page.FileSystems {
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(fsxTagsToMap(fs.Tags))) {
						log.Debug().Interface("filesystem", fs.FileSystemId).Msg("skipping fsx filesystem due to filters")
						continue
					}

					kmsKeyIdStr := ""
					if fs.KmsKeyId != nil {
						kmsKeyIdStr = *fs.KmsKeyId
					}

					args := map[string]*llx.RawData{
						"id":              llx.StringDataPtr(fs.FileSystemId),
						"arn":             llx.StringDataPtr(fs.ResourceARN),
						"type":            llx.StringData(string(fs.FileSystemType)),
						"lifecycle":       llx.StringData(string(fs.Lifecycle)),
						"storageCapacity": llx.IntDataDefault(fs.StorageCapacity, 0),
						"storageType":     llx.StringData(string(fs.StorageType)),
						"kmsKeyId":        llx.StringData(kmsKeyIdStr),
						"vpcId":           llx.StringDataPtr(fs.VpcId),
						"subnetIds":       llx.ArrayData(convert.SliceAnyToInterface(fs.SubnetIds), types.String),
						"tags":            llx.MapData(fsxTagsToMap(fs.Tags), types.String),
						"createdAt":       llx.TimeDataPtr(fs.CreationTime),
						"region":          llx.StringData(regionVal),
					}
					mqlFilesystem, err := CreateResource(a.MqlRuntime, ResourceAwsFsxFilesystem, args)
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

func initAwsFsxFilesystem(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch fsx filesystem")
	}

	// load all fsx filesystems
	obj, err := CreateResource(runtime, ResourceAwsFsx, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	fsxService := obj.(*mqlAwsFsx)
	rawResources := fsxService.GetFileSystems()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		fs := rawResource.(*mqlAwsFsxFilesystem)
		if fs.Arn.Data == arnVal {
			return args, fs, nil
		}
	}
	return nil, nil, errors.New("fsx filesystem does not exist")
}

func (a *mqlAwsFsxFilesystem) encrypted() (bool, error) {
	return a.KmsKeyId.Data != "", nil
}

// ========================
// aws.fsx.cache
// ========================

func (a *mqlAwsFsxCache) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsFsx) caches() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCaches(conn), 5)
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

func (a *mqlAwsFsx) getCaches(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("fsx>getCaches>calling aws with region %s", regionVal)

			svc := conn.Fsx(regionVal)
			ctx := context.Background()
			res := []any{}

			paginator := fsx.NewDescribeFileCachesPaginator(svc, &fsx.DescribeFileCachesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					// Handle "feature not enabled" error for File Cache
					if isBadRequestError(err) {
						log.Debug().Str("region", regionVal).Msg("Amazon File Cache feature not enabled for this account")
						return res, nil
					}
					return nil, err
				}

				for _, cache := range page.FileCaches {
					// Note: FileCache type doesn't have Tags field, skip tag filtering

					lustreConfig, _ := convert.JsonToDict(cache.LustreConfiguration)

					// Convert data repository association IDs to []dict format
					// The actual associations would need to be fetched via DescribeDataRepositoryAssociations
					// For now, we store them as simple dicts with the IDs
					var dataRepoAssocs []any
					for _, assocId := range cache.DataRepositoryAssociationIds {
						dataRepoAssocs = append(dataRepoAssocs, map[string]any{"id": assocId})
					}

					args := map[string]*llx.RawData{
						"id":                         llx.StringDataPtr(cache.FileCacheId),
						"arn":                        llx.StringDataPtr(cache.ResourceARN),
						"lifecycle":                  llx.StringData(string(cache.Lifecycle)),
						"storageCapacity":            llx.IntDataDefault(cache.StorageCapacity, 0),
						"vpcId":                      llx.StringDataPtr(cache.VpcId),
						"subnetIds":                  llx.ArrayData(convert.SliceAnyToInterface(cache.SubnetIds), types.String),
						"lustreConfiguration":        llx.DictData(lustreConfig),
						"dataRepositoryAssociations": llx.ArrayData(dataRepoAssocs, types.Dict),
						"tags":                       llx.MapData(make(map[string]any), types.String), // FileCache doesn't have Tags
					}
					mqlCache, err := CreateResource(a.MqlRuntime, ResourceAwsFsxCache, args)
					if err != nil {
						return nil, err
					}

					res = append(res, mqlCache)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsFsxCache(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch fsx cache")
	}

	// load all fsx caches
	obj, err := CreateResource(runtime, ResourceAwsFsx, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	fsxService := obj.(*mqlAwsFsx)
	rawResources := fsxService.GetCaches()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		cache := rawResource.(*mqlAwsFsxCache)
		if cache.Arn.Data == arnVal {
			return args, cache, nil
		}
	}
	return nil, nil, errors.New("fsx cache does not exist")
}

// ========================
// aws.fsx.backup
// ========================

func (a *mqlAwsFsxBackup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsFsx) backups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getBackups(conn), 5)
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

func (a *mqlAwsFsx) getBackups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("fsx>getBackups>calling aws with region %s", regionVal)

			svc := conn.Fsx(regionVal)
			ctx := context.Background()
			res := []any{}

			paginator := fsx.NewDescribeBackupsPaginator(svc, &fsx.DescribeBackupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, backup := range page.Backups {
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(fsxTagsToMap(backup.Tags))) {
						log.Debug().Interface("backup", backup.BackupId).Msg("skipping fsx backup due to filters")
						continue
					}

					// Get file system ID and type from the FileSystem field if available
					var fileSystemId, fileSystemType string
					if backup.FileSystem != nil {
						if backup.FileSystem.FileSystemId != nil {
							fileSystemId = *backup.FileSystem.FileSystemId
						}
						fileSystemType = string(backup.FileSystem.FileSystemType)
					}

					kmsKeyIdStr := ""
					if backup.KmsKeyId != nil {
						kmsKeyIdStr = *backup.KmsKeyId
					}

					args := map[string]*llx.RawData{
						"backupId":       llx.StringDataPtr(backup.BackupId),
						"arn":            llx.StringDataPtr(backup.ResourceARN),
						"type":           llx.StringData(string(backup.Type)),
						"lifecycle":      llx.StringData(string(backup.Lifecycle)),
						"fileSystemId":   llx.StringData(fileSystemId),
						"fileSystemType": llx.StringData(fileSystemType),
						"kmsKeyId":       llx.StringData(kmsKeyIdStr),
						"createdAt":      llx.TimeDataPtr(backup.CreationTime),
						"tags":           llx.MapData(fsxTagsToMap(backup.Tags), types.String),
					}
					mqlBackup, err := CreateResource(a.MqlRuntime, ResourceAwsFsxBackup, args)
					if err != nil {
						return nil, err
					}

					res = append(res, mqlBackup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsFsxBackup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["backupId"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch fsx backup")
	}

	// load all fsx backups
	obj, err := CreateResource(runtime, ResourceAwsFsx, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	fsxService := obj.(*mqlAwsFsx)
	rawResources := fsxService.GetBackups()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		backup := rawResource.(*mqlAwsFsxBackup)
		if backup.Arn.Data == arnVal {
			return args, backup, nil
		}
	}
	return nil, nil, errors.New("fsx backup does not exist")
}

// ========================
// aws.fsx.volume
// ========================

func (a *mqlAwsFsxVolume) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsFsx) volumes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getVolumes(conn), 5)
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

func (a *mqlAwsFsx) getVolumes(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("fsx>getVolumes>calling aws with region %s", regionVal)

			svc := conn.Fsx(regionVal)
			ctx := context.Background()
			res := []any{}

			paginator := fsx.NewDescribeVolumesPaginator(svc, &fsx.DescribeVolumesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					if isBadRequestError(err) {
						log.Debug().Str("region", regionVal).Msg("Amazon FSx Volumes feature not enabled for this account")
						return res, nil
					}
					return nil, err
				}

				for _, volume := range page.Volumes {
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(fsxTagsToMap(volume.Tags))) {
						log.Debug().Interface("volume", volume.VolumeId).Msg("skipping fsx volume due to filters")
						continue
					}

					svmId := ""
					if volume.OntapConfiguration != nil && volume.OntapConfiguration.StorageVirtualMachineId != nil {
						svmId = *volume.OntapConfiguration.StorageVirtualMachineId
					}

					// ONTAP configuration fields
					var ontapSizeInMegabytes int64
					var ontapJunctionPath string
					var ontapSecurityStyle string
					var ontapStorageEfficiency bool
					var ontapSnapshotPolicy string
					var ontapCopyTagsToBackups bool
					var ontapVolumeStyle string
					if volume.OntapConfiguration != nil {
						if volume.OntapConfiguration.SizeInMegabytes != nil {
							ontapSizeInMegabytes = int64(*volume.OntapConfiguration.SizeInMegabytes)
						}
						if volume.OntapConfiguration.JunctionPath != nil {
							ontapJunctionPath = *volume.OntapConfiguration.JunctionPath
						}
						ontapSecurityStyle = string(volume.OntapConfiguration.SecurityStyle)
						if volume.OntapConfiguration.StorageEfficiencyEnabled != nil {
							ontapStorageEfficiency = *volume.OntapConfiguration.StorageEfficiencyEnabled
						}
						if volume.OntapConfiguration.SnapshotPolicy != nil {
							ontapSnapshotPolicy = *volume.OntapConfiguration.SnapshotPolicy
						}
						if volume.OntapConfiguration.CopyTagsToBackups != nil {
							ontapCopyTagsToBackups = *volume.OntapConfiguration.CopyTagsToBackups
						}
						ontapVolumeStyle = string(volume.OntapConfiguration.VolumeStyle)
					}

					// OpenZFS configuration fields
					var openzfsVolumePath string
					var openzfsParentVolumeId string
					var openzfsReadOnly bool
					var openzfsRecordSizeKiB int64
					var openzfsStorageCapacityQuotaGiB int64
					var openzfsStorageCapacityReservationGiB int64
					var openzfsDataCompressionType string
					var openzfsCopyTagsToSnapshots bool
					if volume.OpenZFSConfiguration != nil {
						if volume.OpenZFSConfiguration.VolumePath != nil {
							openzfsVolumePath = *volume.OpenZFSConfiguration.VolumePath
						}
						if volume.OpenZFSConfiguration.ParentVolumeId != nil {
							openzfsParentVolumeId = *volume.OpenZFSConfiguration.ParentVolumeId
						}
						if volume.OpenZFSConfiguration.ReadOnly != nil {
							openzfsReadOnly = *volume.OpenZFSConfiguration.ReadOnly
						}
						if volume.OpenZFSConfiguration.RecordSizeKiB != nil {
							openzfsRecordSizeKiB = int64(*volume.OpenZFSConfiguration.RecordSizeKiB)
						}
						if volume.OpenZFSConfiguration.StorageCapacityQuotaGiB != nil {
							openzfsStorageCapacityQuotaGiB = int64(*volume.OpenZFSConfiguration.StorageCapacityQuotaGiB)
						}
						if volume.OpenZFSConfiguration.StorageCapacityReservationGiB != nil {
							openzfsStorageCapacityReservationGiB = int64(*volume.OpenZFSConfiguration.StorageCapacityReservationGiB)
						}
						openzfsDataCompressionType = string(volume.OpenZFSConfiguration.DataCompressionType)
						if volume.OpenZFSConfiguration.CopyTagsToSnapshots != nil {
							openzfsCopyTagsToSnapshots = *volume.OpenZFSConfiguration.CopyTagsToSnapshots
						}
					}

					args := map[string]*llx.RawData{
						"__id":                                 llx.StringDataPtr(volume.ResourceARN),
						"id":                                   llx.StringDataPtr(volume.VolumeId),
						"arn":                                  llx.StringDataPtr(volume.ResourceARN),
						"name":                                 llx.StringDataPtr(volume.Name),
						"fileSystemId":                         llx.StringDataPtr(volume.FileSystemId),
						"volumeType":                           llx.StringData(string(volume.VolumeType)),
						"lifecycle":                            llx.StringData(string(volume.Lifecycle)),
						"storageVirtualMachineId":              llx.StringData(svmId),
						"createdAt":                            llx.TimeDataPtr(volume.CreationTime),
						"tags":                                 llx.MapData(fsxTagsToMap(volume.Tags), types.String),
						"region":                               llx.StringData(regionVal),
						"ontapSizeInMegabytes":                 llx.IntData(ontapSizeInMegabytes),
						"ontapJunctionPath":                    llx.StringData(ontapJunctionPath),
						"ontapSecurityStyle":                   llx.StringData(ontapSecurityStyle),
						"ontapStorageEfficiency":               llx.BoolData(ontapStorageEfficiency),
						"ontapSnapshotPolicy":                  llx.StringData(ontapSnapshotPolicy),
						"ontapCopyTagsToBackups":               llx.BoolData(ontapCopyTagsToBackups),
						"ontapVolumeStyle":                     llx.StringData(ontapVolumeStyle),
						"openzfsVolumePath":                    llx.StringData(openzfsVolumePath),
						"openzfsParentVolumeId":                llx.StringData(openzfsParentVolumeId),
						"openzfsReadOnly":                      llx.BoolData(openzfsReadOnly),
						"openzfsRecordSizeKiB":                 llx.IntData(openzfsRecordSizeKiB),
						"openzfsStorageCapacityQuotaGiB":       llx.IntData(openzfsStorageCapacityQuotaGiB),
						"openzfsStorageCapacityReservationGiB": llx.IntData(openzfsStorageCapacityReservationGiB),
						"openzfsDataCompressionType":           llx.StringData(openzfsDataCompressionType),
						"openzfsCopyTagsToSnapshots":           llx.BoolData(openzfsCopyTagsToSnapshots),
					}
					mqlVolume, err := CreateResource(a.MqlRuntime, ResourceAwsFsxVolume, args)
					if err != nil {
						return nil, err
					}

					res = append(res, mqlVolume)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// ========================
// Helper functions
// ========================

func fsxTagsToMap(tags []fsxtypes.Tag) map[string]any {
	tagsMap := make(map[string]any)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}
	return tagsMap
}
