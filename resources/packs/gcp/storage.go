package gcp

import (
	"context"
	"strconv"
	"time"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

func (g *mqlGcpStorage) id() (string, error) {
	return "gcp.storage", nil
}

func (g *mqlGcpStorage) GetBuckets() ([]interface{}, error) {
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

	res := []interface{}{}

	for i := range buckets.Items {
		bucket := buckets.Items[i]

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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.storage.bucket",
			"id", bucket.Id,
			"name", bucket.Name,
			"labels", core.StrMapToInterface(bucket.Labels),
			"location", bucket.Location,
			"locationType", bucket.LocationType,
			"projectNumber", strconv.FormatUint(bucket.ProjectNumber, 10),
			"storageClass", bucket.StorageClass,
			"created", created,
			"updated", updated,
			"iamConfiguration", iamConfigurationDict,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpStorageBucket) id() (string, error) {
	return g.Name()
}

func (g *mqlGcpStorageBucket) GetIamPolicy() ([]interface{}, error) {
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
