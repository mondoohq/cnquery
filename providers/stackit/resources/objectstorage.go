// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlStackitObjectStorageBucketInternal struct {
	retentionFetched bool
	retentionDays    int64
	retentionMode    string
	retentionLock    sync.Mutex
}

func (r *mqlStackitObjectStorage) buckets() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ObjectStorage()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListBucketsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		// A 404 here means the project is not onboarded to Object Storage
		// (the service returns "project.not_found"); treat it as no buckets.
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetBucketsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildBucket(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildBucket(runtime *plugin.Runtime, b *objectstorage.Bucket) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"name":                  llx.StringData(b.GetName()),
		"region":                llx.StringData(b.GetRegion()),
		"urlPathStyle":          llx.StringData(b.GetUrlPathStyle()),
		"urlVirtualHostedStyle": llx.StringData(b.GetUrlVirtualHostedStyle()),
		"objectLockEnabled":     llx.BoolData(b.GetObjectLockEnabled()),
	}
	return CreateResource(runtime, "stackit.objectStorage.bucket", args)
}

func (r *mqlStackitObjectStorageBucket) id() (string, error) {
	return "stackit.objectStorage.bucket/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitObjectStorageBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	v, ok := args["name"]
	if !ok || v == nil {
		return args, nil, nil
	}
	name, ok := v.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.ObjectStorage()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetBucketExecute(bgctx(), c.ProjectID(), c.Region(), name)
	if err != nil {
		return nil, nil, err
	}
	b, ok := resp.GetBucketOk()
	if !ok {
		return args, nil, nil
	}
	res, err := buildBucket(runtime, &b)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// fetchDefaultRetention pulls the bucket's default-retention policy and
// caches it so callers querying both `defaultRetentionDays` and
// `defaultRetentionMode` only hit the API once. A 404 means no default
// retention is set; we cache the zero values so we don't retry.
func (r *mqlStackitObjectStorageBucket) fetchDefaultRetention() (int64, string, error) {
	if r.retentionFetched {
		return r.retentionDays, r.retentionMode, nil
	}
	r.retentionLock.Lock()
	defer r.retentionLock.Unlock()
	if r.retentionFetched {
		return r.retentionDays, r.retentionMode, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.ObjectStorage()
	if err != nil {
		return 0, "", err
	}
	resp, err := client.GetDefaultRetentionExecute(bgctx(), c.ProjectID(), c.Region(), r.Name.Data)
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			r.retentionFetched = true
			return 0, "", nil
		}
		return 0, "", err
	}
	r.retentionDays = resp.GetDays()
	r.retentionMode = string(resp.GetMode())
	r.retentionFetched = true
	return r.retentionDays, r.retentionMode, nil
}

func (r *mqlStackitObjectStorageBucket) defaultRetentionDays() (int64, error) {
	days, _, err := r.fetchDefaultRetention()
	return days, err
}

func (r *mqlStackitObjectStorageBucket) defaultRetentionMode() (string, error) {
	_, mode, err := r.fetchDefaultRetention()
	return mode, err
}
