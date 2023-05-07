package oci

import (
	"context"
	"time"

	"go.mondoo.com/cnquery/resources"

	"github.com/oracle/oci-go-sdk/v65/objectstorage"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
	oci_provider "go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	corePack "go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlOciObjectStorage) id() (string, error) {
	return "oci.objectStorage", nil
}

func (o *mqlOciObjectStorage) GetNamespace() (interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	tenant, err := provider.Tenant(ctx)
	if err != nil {
		return nil, err
	}

	region := *tenant.HomeRegionKey
	client, err := provider.ObjectStorageClient(region)
	if err != nil {
		return "", err
	}

	response, err := client.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
	if err != nil {
		return "", err
	}

	return corePack.ToString(response.Value), nil
}

func (o *mqlOciObjectStorage) GetBuckets() ([]interface{}, error) {
	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	namespace, err := o.GetNamespace()
	if err != nil {
		return nil, err
	}
	namespaceVal := namespace.(string)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(o.getBuckets(provider, namespaceVal), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (o *mqlOciObjectStorage) getBucketsForRegion(ctx context.Context, objectStorageClient *objectstorage.ObjectStorageClient, compartmentID string, namespace string) ([]objectstorage.BucketSummary, error) {
	entries := []objectstorage.BucketSummary{}
	var page *string
	for {
		request := objectstorage.ListBucketsRequest{
			NamespaceName: common.String(namespace),
			CompartmentId: common.String(compartmentID),
			Page:          page,
		}

		response, err := objectStorageClient.ListBuckets(ctx, request)
		if err != nil {
			return nil, err
		}

		entries = append(entries, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return entries, nil
}

func (o *mqlOciObjectStorage) getBuckets(provider *oci_provider.Provider, namespace string) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions(ctx)
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionVal)

			svc, err := provider.ObjectStorageClient(*regionVal.RegionKey)
			if err != nil {
				return nil, err
			}

			var res []interface{}
			buckets, err := o.getBucketsForRegion(ctx, svc, provider.TenantID(), namespace)
			if err != nil {
				return nil, err
			}

			for i := range buckets {
				bucket := buckets[i]

				var created *time.Time
				if bucket.TimeCreated != nil {
					created = &bucket.TimeCreated.Time
				}

				mqlInstance, err := o.MotorRuntime.CreateResource("oci.objectStorage.bucket",
					"namespace", corePack.ToString(bucket.Namespace),
					"name", corePack.ToString(bucket.Name),
					"region", region,
					"created", created,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (o *mqlOciObjectStorageBucket) id() (string, error) {
	namespace, err := o.Namespace()
	if err != nil {
		return "", err
	}

	name, err := o.Name()
	if err != nil {
		return "", err
	}
	return "oci.objectStorage.bucket/" + namespace + "/" + name, nil
}

func (o *mqlOciObjectStorageBucket) getBucketDetails() (*objectstorage.Bucket, error) {
	c, ok := o.MqlResource().Cache.Load("_bucket")
	if ok {
		bucket := c.Data.(*objectstorage.Bucket)
		return bucket, nil
	}

	provider, err := ociProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	region, err := o.Region()
	if err != nil {
		return nil, err
	}
	regionId, err := region.Id()
	if err != nil {
		return nil, err
	}

	client, err := provider.ObjectStorageClient(regionId)
	if err != nil {
		return nil, err
	}

	namespace, err := o.Namespace()
	if err != nil {
		return nil, err
	}

	name, err := o.Name()
	if err != nil {
		return nil, err
	}

	response, err := client.GetBucket(context.Background(), objectstorage.GetBucketRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(name),
	})
	if err != nil {
		return nil, err
	}

	o.MqlResource().Cache.Store("_bucket", &resources.CacheEntry{Data: &response.Bucket})
	return &response.Bucket, nil
}

func (o *mqlOciObjectStorageBucket) GetPublicAccessType() (interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	return string(bucketInfo.PublicAccessType), nil
}

func (o *mqlOciObjectStorageBucket) GetStorageTier() (interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	return string(bucketInfo.StorageTier), nil
}

func (o *mqlOciObjectStorageBucket) GetAutoTiering() (interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	return string(bucketInfo.AutoTiering), nil
}

func (o *mqlOciObjectStorageBucket) GetVersioning() (interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	return string(bucketInfo.Versioning), nil
}

func (o *mqlOciObjectStorageBucket) GetObjectEventsEnabled() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	return *bucketInfo.ObjectEventsEnabled, nil
}

func (o *mqlOciObjectStorageBucket) GetReplicationEnabled() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	return *bucketInfo.ReplicationEnabled, nil
}
