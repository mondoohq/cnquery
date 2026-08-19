// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	madmin "github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7/pkg/sse"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/minio/connection"
)

// mqlMinioBucketInternal caches the per-bucket reads that several fields share.
// The access policy alone backs six fields, so fetching it per field would cost
// six requests for one bucket.
type mqlMinioBucketInternal struct {
	policyLock   sync.Mutex
	policyDoc    string
	policy       *iamPolicy
	policyErr    error
	policyRead   bool
	policyParsed bool

	encLock    sync.Mutex
	encConfig  *sse.Configuration
	encFetched bool

	lockLock     sync.Mutex
	lockFetched  bool
	lockEnabled  string
	lockMode     string
	lockValidity int64
	lockUnit     string

	quotaLock    sync.Mutex
	quotaFetched bool
	quota        madmin.BucketQuota
}

func (a *mqlMinioBucket) conn() *connection.MinioConnection {
	return a.MqlRuntime.Connection.(*connection.MinioConnection)
}

func (a *mqlMinioBucket) id() (string, error) {
	if a.Name.Data == "" {
		return "", errors.New("minio.bucket requires a name")
	}
	return "bucket/" + a.Name.Data, nil
}

// initMinioBucket resolves a bucket named directly, for example
// minio.bucket(name: "my-bucket"). The listing is walked rather than calling
// BucketExists so the creation date comes back with it, and a name that is not
// in the listing is reported as missing rather than turned into a resource
// whose every field would read as unset.
func initMinioBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameArg, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := nameArg.Value.(string)
	if !ok || name == "" {
		return nil, nil, errors.New("minio.bucket requires a name")
	}

	conn := runtime.Connection.(*connection.MinioConnection)
	buckets, err := conn.Client().ListBuckets(context.Background())
	if err != nil {
		return nil, nil, err
	}
	for _, bucket := range buckets {
		if bucket.Name != name {
			continue
		}
		args["createdAt"] = timeData(bucket.CreationDate)
		return args, nil, nil
	}
	return nil, nil, fmt.Errorf("minio.bucket with name %q not found", name)
}

func (a *mqlMinio) buckets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.MinioConnection)
	buckets, err := conn.Client().ListBuckets(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(buckets))
	for _, bucket := range buckets {
		resource, err := CreateResource(a.MqlRuntime, "minio.bucket", map[string]*llx.RawData{
			"__id":      llx.StringData("bucket/" + bucket.Name),
			"name":      llx.StringData(bucket.Name),
			"createdAt": timeData(bucket.CreationDate),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

// location reads the bucket's region. The bucket listing does not carry it:
// MinIO answers ListBuckets with an empty BucketRegion on every entry, so the
// region has to be asked for per bucket.
func (a *mqlMinioBucket) location() (string, error) {
	return a.conn().Client().GetBucketLocation(context.Background(), a.Name.Data)
}

func (a *mqlMinioBucket) tags() (map[string]any, error) {
	tagging, err := a.conn().Client().GetBucketTagging(context.Background(), a.Name.Data)
	if err != nil {
		if isS3ConfigAbsent(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if tagging == nil {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	for k, v := range tagging.ToMap() {
		out[k] = v
	}
	return out, nil
}

func (a *mqlMinioBucket) versioning() (map[string]any, error) {
	config, err := a.conn().Client().GetBucketVersioning(context.Background(), a.Name.Data)
	if err != nil {
		if isS3ConfigAbsent(err) {
			return map[string]any{"Status": "", "MFADelete": ""}, nil
		}
		return nil, err
	}
	return map[string]any{
		"Status":    config.Status,
		"MFADelete": config.MFADelete,
	}, nil
}

func (a *mqlMinioBucket) versioningEnabled() (bool, error) {
	versioning := a.GetVersioning()
	if versioning.Error != nil {
		return false, versioning.Error
	}
	status, _ := versioning.Data["Status"].(string)
	return strings.EqualFold(status, "Enabled"), nil
}

// objectLockConfig reads the bucket's object lock configuration once. A bucket
// created without object locking answers that no configuration exists, which is
// a real answer meaning locking is off.
func (a *mqlMinioBucket) objectLockConfig() (enabled string, mode string, validity int64, unit string, err error) {
	a.lockLock.Lock()
	defer a.lockLock.Unlock()
	if a.lockFetched {
		return a.lockEnabled, a.lockMode, a.lockValidity, a.lockUnit, nil
	}

	lockEnabled, lockMode, lockValidity, lockUnit, err := a.conn().Client().
		GetObjectLockConfig(context.Background(), a.Name.Data)
	if err != nil {
		if !isS3ConfigAbsent(err) {
			return "", "", 0, "", err
		}
		a.lockFetched = true
		return "", "", 0, "", nil
	}

	a.lockFetched = true
	a.lockEnabled = lockEnabled
	if lockMode != nil {
		a.lockMode = string(*lockMode)
	}
	if lockValidity != nil {
		a.lockValidity = int64(*lockValidity)
	}
	if lockUnit != nil {
		a.lockUnit = string(*lockUnit)
	}
	return a.lockEnabled, a.lockMode, a.lockValidity, a.lockUnit, nil
}

func (a *mqlMinioBucket) objectLockEnabled() (bool, error) {
	enabled, _, _, _, err := a.objectLockConfig()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(enabled, "Enabled"), nil
}

func (a *mqlMinioBucket) defaultRetentionMode() (string, error) {
	_, mode, _, _, err := a.objectLockConfig()
	return mode, err
}

func (a *mqlMinioBucket) defaultRetentionValidity() (int64, error) {
	_, _, validity, _, err := a.objectLockConfig()
	return validity, err
}

func (a *mqlMinioBucket) defaultRetentionUnit() (string, error) {
	_, _, _, unit, err := a.objectLockConfig()
	return unit, err
}

// encryptionConfig reads the bucket's default encryption configuration once. A
// bucket with no default encryption answers that the configuration was not
// found, which is a real answer meaning nothing is configured.
func (a *mqlMinioBucket) encryptionConfig() (*sse.Configuration, error) {
	a.encLock.Lock()
	defer a.encLock.Unlock()
	if a.encFetched {
		return a.encConfig, nil
	}

	config, err := a.conn().Client().GetBucketEncryption(context.Background(), a.Name.Data)
	if err != nil {
		if !isS3ConfigAbsent(err) {
			return nil, err
		}
		a.encFetched = true
		return nil, nil
	}
	a.encFetched = true
	a.encConfig = config
	return a.encConfig, nil
}

func (a *mqlMinioBucket) encrypted() (bool, error) {
	config, err := a.encryptionConfig()
	if err != nil {
		return false, err
	}
	return config != nil && len(config.Rules) > 0, nil
}

func (a *mqlMinioBucket) encryptionRules() ([]any, error) {
	config, err := a.encryptionConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return []any{}, nil
	}

	res := make([]any, 0, len(config.Rules))
	for i, rule := range config.Rules {
		resource, err := CreateResource(a.MqlRuntime, "minio.bucket.encryptionRule", map[string]*llx.RawData{
			"__id":         llx.StringData(fmt.Sprintf("bucket/%s/encryptionRule/%d", a.Name.Data, i)),
			"sseAlgorithm": llx.StringData(rule.Apply.SSEAlgorithm),
		})
		if err != nil {
			return nil, err
		}
		mqlRule := resource.(*mqlMinioBucketEncryptionRule)
		mqlRule.cachedKmsKeyName = rule.Apply.KmsMasterKeyID
		res = append(res, resource)
	}
	return res, nil
}

// kmsKey resolves the key the bucket encrypts with by default. SSE-S3 encrypts
// with a server-managed key that has no name, so only an SSE-KMS rule resolves
// to one.
func (a *mqlMinioBucket) kmsKey() (*mqlMinioKmsKey, error) {
	config, err := a.encryptionConfig()
	if err != nil {
		return nil, err
	}
	keyName := ""
	if config != nil {
		for _, rule := range config.Rules {
			if rule.Apply.KmsMasterKeyID != "" {
				keyName = rule.Apply.KmsMasterKeyID
				break
			}
		}
	}
	if keyName == "" {
		a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	resource, err := NewResource(a.MqlRuntime, "minio.kmsKey", map[string]*llx.RawData{
		"name": llx.StringData(keyName),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMinioKmsKey), nil
}

func (a *mqlMinioBucket) bucketQuota() (madmin.BucketQuota, error) {
	a.quotaLock.Lock()
	defer a.quotaLock.Unlock()
	if a.quotaFetched {
		return a.quota, nil
	}

	quota, err := a.conn().Admin().GetBucketQuota(context.Background(), a.Name.Data)
	if err != nil {
		return madmin.BucketQuota{}, err
	}
	a.quotaFetched = true
	a.quota = quota
	return a.quota, nil
}

func (a *mqlMinioBucket) quotaBytes() (int64, error) {
	quota, err := a.bucketQuota()
	if err != nil {
		return 0, err
	}
	return int64(quota.Size), nil
}

func (a *mqlMinioBucket) quotaType() (string, error) {
	quota, err := a.bucketQuota()
	if err != nil {
		return "", err
	}
	return string(quota.Type), nil
}

func (a *mqlMinioBucket) lifecycleRules() ([]any, error) {
	config, err := a.conn().Client().GetBucketLifecycle(context.Background(), a.Name.Data)
	if err != nil {
		if isS3ConfigAbsent(err) {
			return []any{}, nil
		}
		return nil, err
	}
	if config == nil {
		return []any{}, nil
	}

	res := make([]any, 0, len(config.Rules))
	for i, rule := range config.Rules {
		// A rule may be written without an ID, so the index backs the cache key
		// when the ID is empty. Two rules on one bucket would otherwise share a
		// key and the second would be reported as the first.
		key := rule.ID
		if key == "" {
			key = fmt.Sprintf("#%d", i)
		}
		prefix := rule.Prefix
		if prefix == "" {
			prefix = rule.RuleFilter.Prefix
		}

		resource, err := CreateResource(a.MqlRuntime, "minio.bucket.lifecycleRule", map[string]*llx.RawData{
			"__id":                               llx.StringData(fmt.Sprintf("bucket/%s/lifecycleRule/%s", a.Name.Data, key)),
			"id":                                 llx.StringData(rule.ID),
			"status":                             llx.StringData(rule.Status),
			"prefix":                             llx.StringData(prefix),
			"expirationDays":                     llx.IntData(int64(rule.Expiration.Days)),
			"expiredObjectDeleteMarker":          llx.BoolData(bool(rule.Expiration.DeleteMarker)),
			"noncurrentVersionExpirationDays":    llx.IntData(int64(rule.NoncurrentVersionExpiration.NoncurrentDays)),
			"abortIncompleteMultipartUploadDays": llx.IntData(int64(rule.AbortIncompleteMultipartUpload.DaysAfterInitiation)),
			"transitionDays":                     llx.IntData(int64(rule.Transition.Days)),
			"transitionStorageClass":             llx.StringData(rule.Transition.StorageClass),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

func (a *mqlMinioBucket) replicationRules() ([]any, error) {
	config, err := a.conn().Client().GetBucketReplication(context.Background(), a.Name.Data)
	if err != nil {
		if isS3ConfigAbsent(err) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(config.Rules))
	for i, rule := range config.Rules {
		key := rule.ID
		if key == "" {
			key = fmt.Sprintf("#%d", i)
		}
		prefix := rule.Filter.Prefix
		if prefix == "" {
			prefix = rule.Filter.And.Prefix
		}

		resource, err := CreateResource(a.MqlRuntime, "minio.bucket.replicationRule", map[string]*llx.RawData{
			"__id":                             llx.StringData(fmt.Sprintf("bucket/%s/replicationRule/%s", a.Name.Data, key)),
			"id":                               llx.StringData(rule.ID),
			"status":                           llx.StringData(string(rule.Status)),
			"priority":                         llx.IntData(int64(rule.Priority)),
			"destinationBucket":                llx.StringData(rule.Destination.Bucket),
			"destinationStorageClass":          llx.StringData(rule.Destination.StorageClass),
			"prefix":                           llx.StringData(prefix),
			"deleteMarkerReplicationEnabled":   llx.BoolData(strings.EqualFold(string(rule.DeleteMarkerReplication.Status), "Enabled")),
			"existingObjectReplicationEnabled": llx.BoolData(strings.EqualFold(string(rule.ExistingObjectReplication.Status), "Enabled")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

// accessPolicy reads and parses the bucket's access policy once.
//
// MinIO answers a bucket with no policy with an empty body and no error, so an
// empty document is a real "no policy" answer rather than a failed read. A
// document that fails to parse is reported as an error, because silently
// treating an unreadable policy as absent would report an exposed bucket as
// private.
func (a *mqlMinioBucket) accessPolicy() (string, *iamPolicy, error) {
	a.policyLock.Lock()
	defer a.policyLock.Unlock()
	if a.policyParsed {
		return a.policyDoc, a.policy, a.policyErr
	}

	if !a.policyRead {
		document, err := a.conn().Client().GetBucketPolicy(context.Background(), a.Name.Data)
		if err != nil {
			if !isS3ConfigAbsent(err) {
				return "", nil, err
			}
			document = ""
		}
		a.policyRead = true
		a.policyDoc = document
	}

	policy, err := parsePolicyDocument(a.policyDoc)
	a.policyParsed = true
	a.policy = policy
	a.policyErr = err
	return a.policyDoc, a.policy, a.policyErr
}

func (a *mqlMinioBucket) policyDocument() (string, error) {
	document, _, err := a.accessPolicy()
	return document, err
}

func (a *mqlMinioBucket) policyStatements() ([]any, error) {
	_, policy, err := a.accessPolicy()
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return []any{}, nil
	}
	return createPolicyStatements(a.MqlRuntime, "bucket/"+a.Name.Data, policy)
}

// hasAnonymousAccess reports null rather than a confident false when the policy
// could not be read or could not be parsed. A false there is indistinguishable
// from "verified not anonymous", which is the dangerous direction for an
// exposure check on an under-scoped access key.
func (a *mqlMinioBucket) hasAnonymousAccess() (bool, error) {
	_, policy, err := a.accessPolicy()
	if err != nil {
		a.HasAnonymousAccess.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return policyGrantsAnonymousAccess(policy), nil
}

func (a *mqlMinioBucket) hasWildcardPrincipal() (bool, error) {
	_, policy, err := a.accessPolicy()
	if err != nil {
		return false, err
	}
	return policyHasWildcardPrincipal(policy), nil
}

func (a *mqlMinioBucket) hasWildcardAction() (bool, error) {
	_, policy, err := a.accessPolicy()
	if err != nil {
		return false, err
	}
	return policyHasWildcardAction(policy), nil
}

func (a *mqlMinioBucket) enforceSslOnly() (bool, error) {
	_, policy, err := a.accessPolicy()
	if err != nil {
		return false, err
	}
	return policyEnforcesSslOnly(policy), nil
}

// mqlMinioBucketEncryptionRuleInternal carries the key name the rule encrypts
// with, so the typed key reference can be resolved without re-reading the
// bucket's encryption configuration.
type mqlMinioBucketEncryptionRuleInternal struct {
	cachedKmsKeyName string
}

func (a *mqlMinioBucketEncryptionRule) kmsKey() (*mqlMinioKmsKey, error) {
	if a.cachedKmsKeyName == "" {
		a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	resource, err := NewResource(a.MqlRuntime, "minio.kmsKey", map[string]*llx.RawData{
		"name": llx.StringData(a.cachedKmsKeyName),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMinioKmsKey), nil
}
