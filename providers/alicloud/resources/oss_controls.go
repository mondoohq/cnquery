// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"strings"

	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// ossConfigAbsent reports whether an error from a per-bucket configuration call
// means the configuration is simply not set, rather than that the read failed.
//
// Object Storage Service answers an unset configuration with 404 and a
// per-feature NoSuch* code. Only that shape is matched: a 403 from a credential
// without permission, and a transport failure carrying no ServiceError at all,
// both stay errors. Folding them into "not configured" is what would turn an
// unread bucket into a bucket that reports no CORS rules, no retention policy
// and no TLS floor, which is the reading an audit would pass on.
func ossConfigAbsent(err error) bool {
	var svcErr *oss.ServiceError
	if !errors.As(err, &svcErr) {
		return false
	}
	return svcErr.StatusCode == http.StatusNotFound
}

// ossStrings copies a slice of strings into a []any, dropping empty entries so
// an allow list never carries a blank that would read as a configured value.
func ossStrings(in []string) []any {
	res := []any{}
	for _, s := range in {
		if strings.TrimSpace(s) == "" {
			continue
		}
		res = append(res, s)
	}
	return res
}

// policyAllowsPublicAccess reports Object Storage Service's own verdict on
// whether the attached bucket policy opens the bucket. A bucket with no policy
// answers 404, which is not a failure.
func (a *mqlAlicloudOssBucket) policyAllowsPublicAccess() (bool, error) {
	client, err := a.ossClient()
	if err != nil {
		return false, err
	}
	resp, err := client.GetBucketPolicyStatus(context.Background(), &oss.GetBucketPolicyStatusRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return false, nil
		}
		return false, err
	}
	if resp == nil || resp.PolicyStatus == nil || resp.PolicyStatus.IsPublic == nil {
		return false, nil
	}
	return *resp.PolicyStatus.IsPublic, nil
}

// fetchHttpsConfig loads the bucket's TLS version configuration.
func (a *mqlAlicloudOssBucket) fetchHttpsConfig() (*oss.TLS, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketHttpsConfig(context.Background(), &oss.GetBucketHttpsConfigRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil || resp.HttpsConfiguration == nil {
		return nil, nil
	}
	return resp.HttpsConfiguration.TLS, nil
}

func (a *mqlAlicloudOssBucket) tlsEnforced() (bool, error) {
	tls, err := a.fetchHttpsConfig()
	if err != nil || tls == nil || tls.Enable == nil {
		return false, err
	}
	return *tls.Enable, nil
}

func (a *mqlAlicloudOssBucket) tlsVersions() ([]any, error) {
	tls, err := a.fetchHttpsConfig()
	if err != nil || tls == nil {
		return nil, err
	}
	return ossStrings(tls.TLSVersions), nil
}

// fetchWorm loads the bucket's retention policy. A bucket without one answers
// 404, which the SDK documents explicitly for this call.
func (a *mqlAlicloudOssBucket) fetchWorm() (*oss.WormConfiguration, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketWorm(context.Background(), &oss.GetBucketWormRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.WormConfiguration, nil
}

func (a *mqlAlicloudOssBucket) objectLockState() (string, error) {
	worm, err := a.fetchWorm()
	if err != nil || worm == nil {
		return "", err
	}
	return string(worm.State), nil
}

func (a *mqlAlicloudOssBucket) objectLockRetentionDays() (int64, error) {
	worm, err := a.fetchWorm()
	if err != nil || worm == nil || worm.RetentionPeriodInDays == nil {
		return 0, err
	}
	return int64(*worm.RetentionPeriodInDays), nil
}

// fetchReferer loads the bucket's referer restriction.
func (a *mqlAlicloudOssBucket) fetchReferer() (*oss.RefererConfiguration, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketReferer(context.Background(), &oss.GetBucketRefererRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.RefererConfiguration, nil
}

// allowEmptyReferer reports whether a request with no Referer header is
// accepted. Object Storage Service defaults this to true, so an unconfigured
// bucket reports true rather than false: reporting false would claim a
// restriction the bucket does not have.
func (a *mqlAlicloudOssBucket) allowEmptyReferer() (bool, error) {
	cfg, err := a.fetchReferer()
	if err != nil {
		return false, err
	}
	if cfg == nil || cfg.AllowEmptyReferer == nil {
		return true, nil
	}
	return *cfg.AllowEmptyReferer, nil
}

func (a *mqlAlicloudOssBucket) refererAllowList() ([]any, error) {
	cfg, err := a.fetchReferer()
	if err != nil || cfg == nil || cfg.RefererList == nil {
		return nil, err
	}
	return ossStrings(cfg.RefererList.Referers), nil
}

func (a *mqlAlicloudOssBucket) refererDenyList() ([]any, error) {
	cfg, err := a.fetchReferer()
	if err != nil || cfg == nil || cfg.RefererBlacklist == nil {
		return nil, err
	}
	return ossStrings(cfg.RefererBlacklist.Referers), nil
}

// fetchWebsite loads the bucket's static website configuration.
func (a *mqlAlicloudOssBucket) fetchWebsite() (*oss.WebsiteConfiguration, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketWebsite(context.Background(), &oss.GetBucketWebsiteRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.WebsiteConfiguration, nil
}

func (a *mqlAlicloudOssBucket) websiteEnabled() (bool, error) {
	cfg, err := a.fetchWebsite()
	if err != nil {
		return false, err
	}
	return cfg != nil, nil
}

func (a *mqlAlicloudOssBucket) websiteIndexDocument() (string, error) {
	cfg, err := a.fetchWebsite()
	if err != nil || cfg == nil || cfg.IndexDocument == nil || cfg.IndexDocument.Suffix == nil {
		return "", err
	}
	return *cfg.IndexDocument.Suffix, nil
}

func (a *mqlAlicloudOssBucket) websiteErrorDocument() (string, error) {
	cfg, err := a.fetchWebsite()
	if err != nil || cfg == nil || cfg.ErrorDocument == nil || cfg.ErrorDocument.Key == nil {
		return "", err
	}
	return *cfg.ErrorDocument.Key, nil
}

func (a *mqlAlicloudOssBucket) corsRules() ([]any, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketCors(context.Background(), &oss.GetBucketCorsRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return []any{}, nil
		}
		return nil, err
	}
	if resp == nil || resp.CORSConfiguration == nil {
		return []any{}, nil
	}

	res := []any{}
	for i, rule := range resp.CORSConfiguration.CORSRules {
		maxAge := int64(0)
		if rule.MaxAgeSeconds != nil {
			maxAge = *rule.MaxAgeSeconds
		}
		// CORS rules carry no id of their own, so they are keyed by position
		// within the bucket's rule list, which is the order the service applies
		// them in.
		resource, err := CreateResource(a.MqlRuntime, "alicloud.oss.bucket.corsRule", map[string]*llx.RawData{
			"__id":           llx.StringData(a.Name.Data + "/cors/" + itoaInt(i)),
			"allowedOrigins": llx.ArrayData(ossStrings(rule.AllowedOrigins), types.String),
			"allowedMethods": llx.ArrayData(ossStrings(rule.AllowedMethods), types.String),
			"allowedHeaders": llx.ArrayData(ossStrings(rule.AllowedHeaders), types.String),
			"exposeHeaders":  llx.ArrayData(ossStrings(rule.ExposeHeaders), types.String),
			"maxAgeSeconds":  llx.IntData(maxAge),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

func (a *mqlAlicloudOssBucket) replicationRules() ([]any, error) {
	client, err := a.ossClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetBucketReplication(context.Background(), &oss.GetBucketReplicationRequest{
		Bucket: &a.Name.Data,
	})
	if err != nil {
		if ossConfigAbsent(err) {
			return []any{}, nil
		}
		return nil, err
	}
	if resp == nil || resp.ReplicationConfiguration == nil {
		return []any{}, nil
	}

	res := []any{}
	for i, rule := range resp.ReplicationConfiguration.Rules {
		targetBucket, targetLocation := "", ""
		if rule.Destination != nil {
			targetBucket = ossStrVal(rule.Destination.Bucket)
			targetLocation = ossStrVal(rule.Destination.Location)
		}
		prefixes := []any{}
		if rule.PrefixSet != nil {
			// the SDK spells the field Prefixs
			prefixes = ossStrings(rule.PrefixSet.Prefixs)
		}
		kmsKeyID := ""
		if rule.EncryptionConfiguration != nil {
			kmsKeyID = ossStrVal(rule.EncryptionConfiguration.ReplicaKmsKeyID)
		}

		ruleID := ossStrVal(rule.ID)
		key := ruleID
		if key == "" {
			key = itoaInt(i)
		}
		resource, err := CreateResource(a.MqlRuntime, "alicloud.oss.bucket.replicationRule", map[string]*llx.RawData{
			"__id":                        llx.StringData(a.Name.Data + "/replication/" + key),
			"id":                          llx.StringData(ruleID),
			"targetBucket":                llx.StringData(targetBucket),
			"targetLocation":              llx.StringData(targetLocation),
			"status":                      llx.StringData(ossStrVal(rule.Status)),
			"action":                      llx.StringData(ossStrVal(rule.Action)),
			"historicalObjectReplication": llx.StringData(string(rule.HistoricalObjectReplication)),
			"prefixes":                    llx.ArrayData(prefixes, types.String),
			"syncRole":                    llx.StringData(ossStrVal(rule.SyncRole)),
			"replicaKmsKeyId":             llx.StringData(kmsKeyID),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, resource)
	}
	return res, nil
}

// ossPolicyIsPublic decides the policy half of a bucket's public verdict.
//
// serviceVerdict is what GetBucketPolicyStatus reported and is authoritative
// when it could be read. The parsed fallback exists because that call needs its
// own permission: a credential that can read the policy document but not the
// status still gets an answer rather than a silent false.
func ossPolicyIsPublic(serviceVerdict *bool, parsedGrantsAnonymous bool) bool {
	if serviceVerdict != nil {
		return *serviceVerdict
	}
	return parsedGrantsAnonymous
}

// ossStrVal dereferences a *string, returning "" when nil.
func ossStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
