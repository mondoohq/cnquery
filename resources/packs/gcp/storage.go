package gcp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

func (g *mqlGcpProjectStorageService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.storageService/%s", projectId), nil
}

func (g *mqlGcpProjectStorageService) init(args *resources.Args) (*resources.Args, GcpProjectStorageService, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetStorage() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.storageService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectStorageService) GetBuckets() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectID := provider.ResourceID()
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
		}

		var retentionPolicy interface{}
		if bucket.RetentionPolicy != nil {
			retentionPolicy = map[string]interface{}{
				"retentionPeriod": bucket.RetentionPolicy.RetentionPeriod,
				"effectiveTime":   parseTime(bucket.RetentionPolicy.EffectiveTime),
				"isLocked":        bucket.RetentionPolicy.IsLocked,
			}
		}
		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.storageService.bucket",
			"id", bucket.Id,
			"projectId", projectId,
			"name", bucket.Name,
			"labels", core.StrMapToInterface(bucket.Labels),
			"location", bucket.Location,
			"locationType", bucket.LocationType,
			"projectNumber", strconv.FormatUint(bucket.ProjectNumber, 10),
			"storageClass", bucket.StorageClass,
			"created", created,
			"updated", updated,
			"iamConfiguration", iamConfigurationDict,
			"retentionPolicy", retentionPolicy,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (g *mqlGcpProjectStorageServiceBucket) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.storageService.bucket/%s/%s", projectId, id), nil
}

func (g *mqlGcpProjectStorageServiceBucket) init(args *resources.Args) (*resources.Args, GcpProjectStorageServiceBucket, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
		(*args)["location"] = ids.region
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.storageService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	storageSvc := obj.(GcpProjectStorageService)
	buckets, err := storageSvc.Buckets()
	if err != nil {
		return nil, nil, err
	}

	for _, b := range buckets {
		bucket := b.(GcpProjectStorageServiceBucket)
		name, err := bucket.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := bucket.ProjectId()
		if err != nil {
			return nil, nil, err
		}
		location, err := bucket.Location()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] && location == (*args)["location"] {
			return args, bucket, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectStorageServiceBucket) GetIamPolicy() ([]interface{}, error) {
	bucketName, err := g.Name()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
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

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcp.resourcemanager.binding",
			"id", bucketName+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}
