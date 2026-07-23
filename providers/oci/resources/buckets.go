// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
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
	if tenant.HomeRegionKey == nil {
		return "", errors.New("tenancy has no home region configured")
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

	return ociRunRegionPool(o.getBuckets(conn, namespace, list.Data))
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
					"namespace":     llx.StringDataPtr(bucket.Namespace),
					"name":          llx.StringDataPtr(bucket.Name),
					"compartmentID": llx.StringDataPtr(bucket.CompartmentId),
					"region":        llx.ResourceData(regionResource, "oci.region"),
					"created":       llx.TimeDataPtr(created),
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
	lock    sync.Mutex
	fetched atomic.Bool
	bucket  *objectstorage.Bucket
}

func (o *mqlOciObjectStorageBucket) id() (string, error) {
	return "oci.objectStorage.bucket/" + o.Namespace.Data + "/" + o.Name.Data, nil
}

func initOciObjectStorageBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// There is deliberately no id-only fast path. OCI keys GetBucket on
	// namespace+name, not on the bucket OCID, so an id alone cannot be
	// resolved - and the resulting resource would take the cache key
	// "oci.objectStorage.bucket//" (both name parts empty), which every
	// id-only bucket would share.

	// When cnspec scans a discovered oci-objectstorage-bucket asset the only
	// context we have is the Conf.PlatformId. Parse out namespace/name so the
	// singular `oci.objectStorage.bucket` resolves to that specific bucket
	// without the policy having to pass explicit args.
	if ociArgString(args, "namespace") == "" && ociArgString(args, "name") == "" {
		if conn, ok := runtime.Connection.(*connection.OciConnection); ok && conn.Conf != nil && conn.Conf.PlatformId != "" {
			if parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId); ok &&
				parsed.service == "objectstorage" && parsed.objectType == "bucket" {
				// Bucket platform ids encode "<namespace>/<name>".
				if slash := strings.IndexByte(parsed.id, '/'); slash > 0 && slash < len(parsed.id)-1 {
					if args == nil {
						args = map[string]*llx.RawData{}
					}
					args["namespace"] = llx.StringData(parsed.id[:slash])
					args["name"] = llx.StringData(parsed.id[slash+1:])
					if parsed.region != "" && parsed.region != "unknown" {
						// region is a typed oci.region resource; we have only
						// its id (the region key). Create the reference so
						// subsequent accessors can use it.
						regionRes, err := NewResource(runtime, "oci.region", map[string]*llx.RawData{
							"id": llx.StringData(parsed.region),
						})
						if err == nil {
							args["region"] = llx.ResourceData(regionRes, "oci.region")
						}
					}
				}
			}
		}
	}

	obj, err := CreateResource(runtime, "oci.objectStorage.bucket", args)
	if err != nil {
		return nil, nil, err
	}
	bucket := obj.(*mqlOciObjectStorageBucket)

	// getBucketDetails sets Id on the resource itself. Writing args["id"] here
	// would be a no-op: NewResource discards the returned args whenever the
	// init also returns a resource.
	if _, err := bucket.getBucketDetails(); err != nil {
		return nil, nil, err
	}

	return args, bucket, nil
}

// getBucketDetails lazily fetches the full bucket, which carries the fields
// ListBuckets omits (public access, versioning, encryption, counts). Sixteen
// accessors share it, and the runtime resolves them concurrently, so the fetch
// is guarded rather than racing sixteen identical GetBucket calls.
func (o *mqlOciObjectStorageBucket) getBucketDetails() (*objectstorage.Bucket, error) {
	if o.fetched.Load() {
		return o.bucket, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.fetched.Load() {
		return o.bucket, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	region := o.GetRegion()
	if region.Error != nil {
		return nil, region.Error
	}

	if region.Data == nil {
		return nil, errors.New("oci.objectStorage.bucket: region is required to fetch bucket details")
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
		Fields: []objectstorage.GetBucketFieldsEnum{
			objectstorage.GetBucketFieldsApproximatecount,
			objectstorage.GetBucketFieldsApproximatesize,
			objectstorage.GetBucketFieldsAutotiering,
		},
	})
	if err != nil {
		return nil, err
	}

	o.bucket = &response.Bucket
	// ListBuckets returns a BucketSummary, which carries no Id at all, so the
	// bucket OCID is only knowable from this call. Populate it here so both the
	// collection path and the single-bucket init resolve `id` identically.
	if o.bucket.Id != nil {
		o.Id = plugin.TValue[string]{Data: *o.bucket.Id, State: plugin.StateIsSet}
	}
	o.fetched.Store(true)
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
	if bucketInfo.ObjectEventsEnabled == nil {
		return false, nil
	}
	return *bucketInfo.ObjectEventsEnabled, nil
}

func (o *mqlOciObjectStorageBucket) replicationEnabled() (bool, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return false, err
	}
	if bucketInfo.ReplicationEnabled == nil {
		return false, nil
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

func (o *mqlOciObjectStorageBucket) objectLifecyclePolicyEtag() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	if bucketInfo.ObjectLifecyclePolicyEtag == nil {
		return "", nil
	}
	return *bucketInfo.ObjectLifecyclePolicyEtag, nil
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

func (o *mqlOciObjectStorageBucket) createdBy() (string, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return "", err
	}
	if bucketInfo.CreatedBy == nil {
		return "", nil
	}
	return *bucketInfo.CreatedBy, nil
}

func (o *mqlOciObjectStorageBucket) createdByUser() (*mqlOciIdentityUser, error) {
	bucketInfo, err := o.getBucketDetails()
	if err != nil {
		return nil, err
	}
	createdBy := stringValue(bucketInfo.CreatedBy)
	if !strings.HasPrefix(createdBy, "ocid1.user.") {
		o.CreatedByUser.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(o.MqlRuntime, "oci.identity.user", map[string]*llx.RawData{
		"id": llx.StringData(createdBy),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciIdentityUser), nil
}

func (o *mqlOciObjectStorageBucket) retentionRules() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	region := o.GetRegion()
	if region.Error != nil {
		return nil, region.Error
	}
	if region.Data == nil {
		return nil, errors.New("oci.objectStorage.bucket: region is required")
	}

	client, err := conn.ObjectStorageClient(region.Data.Id.Data)
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

	ctx := context.Background()
	var rules []objectstorage.RetentionRuleSummary
	var page *string
	for {
		response, err := client.ListRetentionRules(ctx, objectstorage.ListRetentionRulesRequest{
			NamespaceName: common.String(namespace.Data),
			BucketName:    common.String(name.Data),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		rules = append(rules, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(rules))
	for i := range rules {
		r := rules[i]

		var created *time.Time
		if r.TimeCreated != nil {
			created = &r.TimeCreated.Time
		}
		var timeModified *time.Time
		if r.TimeModified != nil {
			timeModified = &r.TimeModified.Time
		}
		var timeRuleLocked *time.Time
		if r.TimeRuleLocked != nil {
			timeRuleLocked = &r.TimeRuleLocked.Time
		}

		var durationAmount int64
		var durationTimeUnit string
		if r.Duration != nil {
			durationAmount = int64Value(r.Duration.TimeAmount)
			durationTimeUnit = string(r.Duration.TimeUnit)
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.objectStorage.retentionRule", map[string]*llx.RawData{
			"id":               llx.StringDataPtr(r.Id),
			"name":             llx.StringDataPtr(r.DisplayName),
			"durationAmount":   llx.IntData(durationAmount),
			"durationTimeUnit": llx.StringData(durationTimeUnit),
			"timeRuleLocked":   llx.TimeDataPtr(timeRuleLocked),
			"created":          llx.TimeDataPtr(created),
			"timeModified":     llx.TimeDataPtr(timeModified),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciObjectStorageRetentionRule) id() (string, error) {
	return "oci.objectStorage.retentionRule/" + o.Id.Data, nil
}

func (o *mqlOciObjectStorageBucket) preauthenticatedRequests() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	region := o.GetRegion()
	if region.Error != nil {
		return nil, region.Error
	}
	if region.Data == nil {
		return nil, errors.New("oci.objectStorage.bucket: region is required")
	}

	client, err := conn.ObjectStorageClient(region.Data.Id.Data)
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

	ctx := context.Background()
	var pars []objectstorage.PreauthenticatedRequestSummary
	var page *string
	for {
		response, err := client.ListPreauthenticatedRequests(ctx, objectstorage.ListPreauthenticatedRequestsRequest{
			NamespaceName: common.String(namespace.Data),
			BucketName:    common.String(name.Data),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		pars = append(pars, response.Items...)
		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(pars))
	for i := range pars {
		p := pars[i]

		var created *time.Time
		if p.TimeCreated != nil {
			created = &p.TimeCreated.Time
		}
		var timeExpires *time.Time
		if p.TimeExpires != nil {
			timeExpires = &p.TimeExpires.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.objectStorage.preauthenticatedRequest", map[string]*llx.RawData{
			"id":                  llx.StringDataPtr(p.Id),
			"name":                llx.StringDataPtr(p.Name),
			"accessType":          llx.StringData(string(p.AccessType)),
			"objectName":          llx.StringDataPtr(p.ObjectName),
			"bucketListingAction": llx.StringData(string(p.BucketListingAction)),
			"timeExpires":         llx.TimeDataPtr(timeExpires),
			"created":             llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciObjectStoragePreauthenticatedRequest) id() (string, error) {
	return "oci.objectStorage.preauthenticatedRequest/" + o.Id.Data, nil
}
