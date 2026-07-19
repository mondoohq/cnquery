// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"sync/atomic"

	tea "github.com/alibabacloud-go/tea/tea"
	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

func (r *mqlAlicloudOss) id() (string, error) {
	return "alicloud.oss", nil
}

// buckets lists every bucket owned by the account. ListBuckets returns buckets
// from all regions in a single (paginated) call, so it is issued once against
// the connection's default-region client.
func (r *mqlAlicloudOss) buckets() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.OssClient(conn.Region())
	if err != nil {
		return nil, err
	}

	res := []any{}
	var marker *string
	for {
		resp, err := client.ListBuckets(context.Background(), &oss.ListBucketsRequest{
			Marker:  marker,
			MaxKeys: 1000,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for i := range resp.Buckets {
			b := resp.Buckets[i]
			name := tea.StringValue(b.Name)
			region := tea.StringValue(b.Region)
			// A bucket without a resolvable region falls back to the
			// connection's default region so the detail-client build succeeds.
			if region == "" {
				region = conn.Region()
			}

			bucket, err := CreateResource(r.MqlRuntime, "alicloud.oss.bucket", map[string]*llx.RawData{
				"__id":             llx.StringData(name),
				"name":             llx.StringData(name),
				"region":           llx.StringData(region),
				"location":         llx.StringDataPtr(b.Location),
				"storageClass":     llx.StringDataPtr(b.StorageClass),
				"creationDate":     llx.TimeDataPtr(b.CreationDate),
				"intranetEndpoint": llx.StringDataPtr(b.IntranetEndpoint),
				"extranetEndpoint": llx.StringDataPtr(b.ExtranetEndpoint),
				"resourceGroupId":  llx.StringDataPtr(b.ResourceGroupId),
			})
			if err != nil {
				return nil, err
			}
			mqlBucket := bucket.(*mqlAlicloudOssBucket)
			mqlBucket.name = name
			mqlBucket.region = region
			res = append(res, bucket)
		}

		if !resp.IsTruncated || resp.NextMarker == nil {
			break
		}
		marker = resp.NextMarker
	}

	return res, nil
}

// mqlAlicloudOssBucketInternal caches the values needed to build a per-bucket
// OSS client and memoizes the two detail calls (GetBucketInfo and
// GetBucketEncryption) that back more than one accessor.
type mqlAlicloudOssBucketInternal struct {
	name   string
	region string

	infoLock   sync.Mutex
	infoLoaded atomic.Bool
	info       *oss.BucketInfo

	encLock   sync.Mutex
	encLoaded atomic.Bool
	encRule   *oss.ApplyServerSideEncryptionByDefault
}

func (a *mqlAlicloudOssBucket) id() (string, error) {
	return a.Name.Data, nil
}

// ossClient builds an OSS client bound to the bucket's own region so the
// per-bucket detail APIs address the correct endpoint.
func (a *mqlAlicloudOssBucket) ossClient() (*oss.Client, error) {
	conn := a.MqlRuntime.Connection.(*connection.AlicloudConnection)
	return conn.OssClient(a.region)
}

func (a *mqlAlicloudOssBucket) acl() (string, error) {
	client, err := a.ossClient()
	if err != nil {
		return "", err
	}
	resp, err := client.GetBucketAcl(context.Background(), &oss.GetBucketAclRequest{Bucket: &a.name})
	if err != nil {
		// tolerate access-denied / transient errors on this optional detail call
		return "", nil
	}
	if resp == nil || resp.ACL == nil {
		return "", nil
	}
	return *resp.ACL, nil
}

func (a *mqlAlicloudOssBucket) versioning() (string, error) {
	client, err := a.ossClient()
	if err != nil {
		return "", err
	}
	resp, err := client.GetBucketVersioning(context.Background(), &oss.GetBucketVersioningRequest{Bucket: &a.name})
	if err != nil {
		return "", nil
	}
	if resp == nil || resp.VersionStatus == nil {
		return "", nil
	}
	return *resp.VersionStatus, nil
}

// fetchEncryption memoizes the default server-side encryption rule. A missing
// rule (no encryption configured) or an access error both resolve to a cached
// nil so encryption and sseAlgorithm agree and neither re-calls the API.
func (a *mqlAlicloudOssBucket) fetchEncryption() (*oss.ApplyServerSideEncryptionByDefault, error) {
	if a.encLoaded.Load() {
		return a.encRule, nil
	}
	a.encLock.Lock()
	defer a.encLock.Unlock()
	if a.encLoaded.Load() {
		return a.encRule, nil
	}

	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketEncryption(context.Background(), &oss.GetBucketEncryptionRequest{Bucket: &a.name})
	if err != nil {
		// no encryption rule configured, or access denied
		a.encRule = nil
		a.encLoaded.Store(true)
		return nil, nil
	}
	if resp != nil && resp.ServerSideEncryptionRule != nil {
		a.encRule = resp.ServerSideEncryptionRule.ApplyServerSideEncryptionByDefault
	}
	a.encLoaded.Store(true)
	return a.encRule, nil
}

func (a *mqlAlicloudOssBucket) encryption() (any, error) {
	rule, err := a.fetchEncryption()
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, nil
	}
	return convert.JsonToDict(rule)
}

func (a *mqlAlicloudOssBucket) sseAlgorithm() (string, error) {
	rule, err := a.fetchEncryption()
	if err != nil {
		return "", err
	}
	if rule == nil || rule.SSEAlgorithm == nil {
		return "", nil
	}
	return *rule.SSEAlgorithm, nil
}

func (a *mqlAlicloudOssBucket) logging() (any, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketLogging(context.Background(), &oss.GetBucketLoggingRequest{Bucket: &a.name})
	if err != nil {
		return nil, nil
	}
	if resp == nil || resp.BucketLoggingStatus == nil || resp.BucketLoggingStatus.LoggingEnabled == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.BucketLoggingStatus.LoggingEnabled)
}

func (a *mqlAlicloudOssBucket) policy() (string, error) {
	client, err := a.ossClient()
	if err != nil {
		return "", err
	}
	resp, err := client.GetBucketPolicy(context.Background(), &oss.GetBucketPolicyRequest{Bucket: &a.name})
	if err != nil {
		// most buckets have no policy (NoSuchBucketPolicy)
		return "", nil
	}
	if resp == nil {
		return "", nil
	}
	return resp.Body, nil
}

func (a *mqlAlicloudOssBucket) tags() (map[string]any, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketTags(context.Background(), &oss.GetBucketTagsRequest{Bucket: &a.name})
	if err != nil {
		return nil, nil
	}
	res := map[string]any{}
	if resp != nil && resp.Tagging != nil && resp.Tagging.TagSet != nil {
		for _, t := range resp.Tagging.TagSet.Tags {
			if t.Key == nil {
				continue
			}
			res[*t.Key] = tea.StringValue(t.Value)
		}
	}
	return res, nil
}

func (a *mqlAlicloudOssBucket) publicAccessBlock() (any, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketPublicAccessBlock(context.Background(), &oss.GetBucketPublicAccessBlockRequest{Bucket: &a.name})
	if err != nil {
		return nil, nil
	}
	if resp == nil || resp.PublicAccessBlockConfiguration == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.PublicAccessBlockConfiguration)
}

// fetchInfo memoizes GetBucketInfo, which backs bucketInfo and the flattened
// transferAcceleration, crossRegionReplication, dataRedundancyType, and
// blockPublicAccess accessors.
func (a *mqlAlicloudOssBucket) fetchInfo() (*oss.BucketInfo, error) {
	if a.infoLoaded.Load() {
		return a.info, nil
	}
	a.infoLock.Lock()
	defer a.infoLock.Unlock()
	if a.infoLoaded.Load() {
		return a.info, nil
	}

	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketInfo(context.Background(), &oss.GetBucketInfoRequest{Bucket: &a.name})
	if err != nil || resp == nil {
		a.info = nil
		a.infoLoaded.Store(true)
		return nil, nil
	}
	a.info = &resp.BucketInfo
	a.infoLoaded.Store(true)
	return a.info, nil
}

func (a *mqlAlicloudOssBucket) bucketInfo() (any, error) {
	info, err := a.fetchInfo()
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return convert.JsonToDict(info)
}

func (a *mqlAlicloudOssBucket) transferAcceleration() (string, error) {
	info, err := a.fetchInfo()
	if err != nil {
		return "", err
	}
	if info == nil || info.TransferAcceleration == nil {
		return "", nil
	}
	return *info.TransferAcceleration, nil
}

func (a *mqlAlicloudOssBucket) crossRegionReplication() (string, error) {
	info, err := a.fetchInfo()
	if err != nil {
		return "", err
	}
	if info == nil || info.CrossRegionReplication == nil {
		return "", nil
	}
	return *info.CrossRegionReplication, nil
}

func (a *mqlAlicloudOssBucket) dataRedundancyType() (string, error) {
	info, err := a.fetchInfo()
	if err != nil {
		return "", err
	}
	if info == nil || info.DataRedundancyType == nil {
		return "", nil
	}
	return *info.DataRedundancyType, nil
}

func (a *mqlAlicloudOssBucket) blockPublicAccess() (bool, error) {
	info, err := a.fetchInfo()
	if err != nil {
		return false, err
	}
	if info == nil || info.BlockPublicAccess == nil {
		return false, nil
	}
	return *info.BlockPublicAccess, nil
}
