// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

func (g *mqlGcpProjectStorageService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("gcp.project.storageService/%s", projectId), nil
}

func initGcpProjectStorageService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)
	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) storage() (*mqlGcpProjectStorageService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.storageService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectStorageService), nil
}

func (g *mqlGcpProjectStorageService) buckets() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectID := conn.ResourceID()
	buckets, err := storageSvc.Buckets.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(buckets.Items))
	for _, bucket := range buckets.Items {
		created := parseTime(bucket.TimeCreated)
		updated := parseTime(bucket.Updated)

		iamConfigurationDict := map[string]interface{}{}

		if bucket.IamConfiguration != nil {
			iamConfiguration := bucket.IamConfiguration

			if iamConfiguration.BucketPolicyOnly != nil {
				var parsedLockTime time.Time
				if iamConfiguration.BucketPolicyOnly.LockedTime != "" {
					parsedLockTime, err = time.Parse(time.RFC3339, iamConfiguration.BucketPolicyOnly.LockedTime)
					if err != nil {
						return nil, err
					}
				}

				iamConfigurationDict["BucketPolicyOnly"] = map[string]interface{}{
					"enabled":    iamConfiguration.BucketPolicyOnly.Enabled,
					"lockedTime": parsedLockTime,
				}
			}

			if iamConfiguration.UniformBucketLevelAccess != nil {
				var parsedLockTime time.Time
				if iamConfiguration.UniformBucketLevelAccess.LockedTime != "" {
					parsedLockTime, err = time.Parse(time.RFC3339, iamConfiguration.UniformBucketLevelAccess.LockedTime)
					if err != nil {
						return nil, err
					}
				}

				iamConfigurationDict["UniformBucketLevelAccess"] = map[string]interface{}{
					"enabled":    iamConfiguration.UniformBucketLevelAccess.Enabled,
					"lockedTime": parsedLockTime,
				}
			}

			iamConfigurationDict["publicAccessPrevention"] = iamConfiguration.PublicAccessPrevention
		}

		var retentionPolicy interface{}
		if bucket.RetentionPolicy != nil {
			retentionPolicy = map[string]interface{}{
				"retentionPeriod": bucket.RetentionPolicy.RetentionPeriod,
				"effectiveTime":   parseTime(bucket.RetentionPolicy.EffectiveTime),
				"isLocked":        bucket.RetentionPolicy.IsLocked,
			}
		}
		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.storageService.bucket", map[string]*llx.RawData{
			"id":               llx.StringData(bucket.Id),
			"projectId":        llx.StringData(projectId),
			"name":             llx.StringData(bucket.Name),
			"labels":           llx.MapData(convert.MapToInterfaceMap(bucket.Labels), types.String),
			"location":         llx.StringData(bucket.Location),
			"locationType":     llx.StringData(bucket.LocationType),
			"projectNumber":    llx.StringData(strconv.FormatUint(bucket.ProjectNumber, 10)),
			"storageClass":     llx.StringData(bucket.StorageClass),
			"created":          llx.TimeDataPtr(created),
			"updated":          llx.TimeDataPtr(updated),
			"iamConfiguration": llx.DictData(iamConfigurationDict),
			"retentionPolicy":  llx.DictData(retentionPolicy),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (g *mqlGcpProjectStorageServiceBucket) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("gcp.project.storageService.bucket/%s/%s", projectId, id), nil
}

func initGcpProjectStorageServiceBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
			args["location"] = llx.StringData(ids.region)
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.storageService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	storageSvc := obj.(*mqlGcpProjectStorageService)
	buckets := storageSvc.GetBuckets()
	if buckets.Error != nil {
		return nil, nil, buckets.Error
	}

	for _, b := range buckets.Data {
		bucket := b.(*mqlGcpProjectStorageServiceBucket)

		if bucket.Name.Error != nil {
			return nil, nil, bucket.Name.Error
		}
		name := bucket.Name.Data

		if bucket.ProjectId.Error != nil {
			return nil, nil, bucket.ProjectId.Error
		}
		projectId := bucket.ProjectId.Data

		if bucket.Location.Error != nil {
			return nil, nil, bucket.Location.Error
		}
		location := bucket.Location.Data

		if name == args["name"].Value.(string) && projectId == args["projectId"].Value.(string) && location == args["location"].Value.(string) {
			return args, bucket, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectStorageServiceBucket) iamPolicy() ([]interface{}, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	bucketName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storeSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := storeSvc.Buckets.GetIamPolicy(bucketName).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":      llx.StringData(bucketName + "-" + strconv.Itoa(i)),
			"role":    llx.StringData(b.Role),
			"members": llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}
