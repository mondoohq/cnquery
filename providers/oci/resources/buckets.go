// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
)

func (e *mqlOciObjectStorage) id() (string, error) {
	return "oci.objectStorage", nil
}

func (o *mqlOciObjectStorage) namespace() (string, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ctx := context.Background()
	tenant, err := conn.Tenant(ctx)
	if err != nil {
		return "", err
	}

	region := *tenant.HomeRegionKey
	client, err := conn.ObjectStorageClient(region)
	if err != nil {
		return "", err
	}

	response, err := client.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
	if err != nil {
		return "", err
	}

	if response.Value == nil {
		return "", nil
	} else {
		return *response.Value, nil
	}
}

func (o *mqlOciObjectStorage) buckets() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// fetch regions
	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	// fetch buckets
	namespace, err := o.namespace()
	if err != nil {
		return nil, err
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getBuckets(conn, namespace, list.Data), 5)
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

func (o *mqlOciObjectStorage) getBuckets(conn *connection.OciConnection, namespace string, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)

	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			svc, err := conn.ObjectStorageClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			var res []any
			buckets, err := o.getBucketsForRegion(ctx, svc, conn.TenantID(), namespace)
			if err != nil {
				return nil, err
			}

			for i := range buckets {
				bucket := buckets[i]

				var created *time.Time
				if bucket.TimeCreated != nil {
					created = &bucket.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.objectStorage.bucket", map[string]*llx.RawData{
					"namespace": llx.StringDataPtr(bucket.Namespace),
					"name":      llx.StringDataPtr(bucket.Name),
					"region":    llx.ResourceData(regionResource, "oci.region"),
					"created":   llx.TimeDataPtr(created),
				})
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

type mqlOciObjectStorageBucketInternal struct {
	bucket *objectstorage.Bucket
}

func (o *mqlOciObjectStorageBucket) id() (string, error) {
	return "oci.objectStorage.bucket/" + o.Namespace.Data + "/" + o.Name.Data, nil
}

func initOciObjectStorageBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Check if id is already populated
	if id, ok := args["id"]; ok && id.Value != nil {
		if idStr, ok := id.Value.(string); ok && idStr != "" {
			return args, nil, nil
		}
	}

	obj, err := CreateResource(runtime, "oci.objectStorage.bucket", args)
	if err != nil {
		return nil, nil, err
	}
	bucket := obj.(*mqlOciObjectStorageBucket)

	// Fetch bucket details to populate the id field
	bucketDetails, err := bucket.getBucketDetails()
	if err != nil {
		return nil, nil, err
	}

	if bucketDetails.Id != nil {
		args["id"] = llx.StringData(*bucketDetails.Id)
	}

	return args, bucket, nil
}

func (o *mqlOciObjectStorageBucket) getBucketDetails() (*objectstorage.Bucket, error) {
	if o.bucket != nil {
		return o.bucket, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	region := o.GetRegion()
	if region.Error != nil {
		return nil, region.Error
	}

	r := region.Data
	client, err := conn.ObjectStorageClient(r.Id.Data)
	if err != nil {
		return nil, err
	}

	namespace := o.GetNamespace()
	if namespace.Error != nil {
		return nil, namespace.Error
	}

	name := o.GetName()
	if name.Error != nil {
		return nil, name.Error
	}

	response, err := client.GetBucket(context.Background(), objectstorage.GetBucketRequest{
		NamespaceName: common.String(namespace.Data),
		BucketName:    common.String(name.Data),
	})
	if err != nil {
		return nil, err
	}

	o.bucket = &response.Bucket
	return o.bucket, nil
}

func (o *mqlOciObjectStorageBucket) publicAccessType() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	return string(bucketInfo.PublicAccessType), nil
}

func (o *mqlOciObjectStorageBucket) storageTier() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	return string(bucketInfo.StorageTier), nil
}

func (o *mqlOciObjectStorageBucket) autoTiering() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	return string(bucketInfo.AutoTiering), nil
}

func (o *mqlOciObjectStorageBucket) versioning() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	return string(bucketInfo.Versioning), nil
}

func (o *mqlOciObjectStorageBucket) objectEventsEnabled() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	return *bucketInfo.ObjectEventsEnabled, nil
}

func (o *mqlOciObjectStorageBucket) replicationEnabled() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	return *bucketInfo.ReplicationEnabled, nil
}

func (o *mqlOciObjectStorageBucket) isReadOnly() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	if bucketInfo.IsReadOnly == nil {
		return false, nil
	}
	return *bucketInfo.IsReadOnly, nil
}

func (o *mqlOciObjectStorageBucket) etag() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	if bucketInfo.Etag == nil {
		return "", nil
	}
	return *bucketInfo.Etag, nil
}

func (o *mqlOciObjectStorageBucket) kmsKeyId() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	if bucketInfo.KmsKeyId == nil {
		return "", nil
	}
	return *bucketInfo.KmsKeyId, nil
}

func (o *mqlOciObjectStorageBucket) approximateCount() (int64, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return 0, err
	}
	if bucketInfo.ApproximateCount == nil {
		return 0, nil
	}
	return *bucketInfo.ApproximateCount, nil
}

func (o *mqlOciObjectStorageBucket) approximateSize() (int64, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return 0, err
	}
	if bucketInfo.ApproximateSize == nil {
		return 0, nil
	}
	return *bucketInfo.ApproximateSize, nil
}

func (o *mqlOciObjectStorageBucket) freeformTags() (map[string]interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	tags := make(map[string]interface{})
	for k, v := range bucketInfo.FreeformTags {
		tags[k] = v
	}
	return tags, nil
}

func (o *mqlOciObjectStorageBucket) definedTags() (map[string]interface{}, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	tags := make(map[string]interface{})
	for k, v := range bucketInfo.DefinedTags {
		tags[k] = v
	}
	return tags, nil
}
