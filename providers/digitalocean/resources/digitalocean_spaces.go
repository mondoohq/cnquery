// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/digitalocean/connection"
	"go.mondoo.com/mql/v13/types"
)

// knownSpacesRegions enumerates the DigitalOcean regions that host
// Spaces. The list grows when DO opens new regions — keep it in sync
// with https://docs.digitalocean.com/products/platform/availability-matrix/
var knownSpacesRegions = []string{
	"nyc3", "sfo2", "sfo3", "ams3", "sgp1", "fra1", "syd1", "tor1", "blr1",
}

// regionConcurrency caps the parallel ListBuckets sweep across Spaces
// regions. Most accounts use a handful of regions, so a small number
// is fine.
const regionConcurrency = 5

// bucketConcurrency caps the parallel per-bucket detail fan-out.
// Each bucket triggers up to 7 sequential S3-compatible API calls
// (access block, ACL, encryption, versioning, policy, CORS,
// lifecycle); running them across multiple buckets in parallel is
// the main win.
const bucketConcurrency = 8

func (r *mqlDigitaloceanSpacesBucket) id() (string, error) {
	return "digitalocean.spacesBucket/" + r.Region.Data + "/" + r.Name.Data, nil
}

// bucketRef carries the (region, client, summary) tuple from the
// per-region ListBuckets pass into the per-bucket detail-fetch pass.
type bucketRef struct {
	region string
	client *s3.Client
	b      s3types.Bucket
}

func (r *mqlDigitalocean) spacesBuckets() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DigitaloceanConnection)
	if _, _, ok := conn.SpacesCredentials(); !ok {
		// Spaces credentials are optional — return an empty list so
		// the rest of the provider stays usable without them.
		return []interface{}{}, nil
	}

	regions := []string{conn.SpacesRegion()}
	if regions[0] == "" {
		regions = knownSpacesRegions
	}

	refs, err := listSpacesBucketRefs(conn, regions)
	if err != nil {
		return nil, err
	}
	return fetchSpacesBucketDetails(r, refs)
}

// listSpacesBucketRefs lists buckets across the requested regions in
// parallel and returns the merged refs slice. ListBuckets is
// region-scoped on DigitalOcean Spaces (each region's endpoint
// returns only that region's buckets), so the refs are disjoint.
func listSpacesBucketRefs(conn *connection.DigitaloceanConnection, regions []string) ([]bucketRef, error) {
	ctx := context.Background()
	var (
		mu   sync.Mutex
		refs []bucketRef
	)
	jobs := make([]*jobpool.Job, 0, len(regions))
	for _, region := range regions {
		jobs = append(jobs, jobpool.NewJob(func() (jobpool.JobResult, error) {
			client, err := conn.SpacesClient(region)
			if err != nil {
				return nil, err
			}
			out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
			if err != nil {
				return nil, err
			}
			local := make([]bucketRef, 0, len(out.Buckets))
			for _, b := range out.Buckets {
				local = append(local, bucketRef{region: region, client: client, b: b})
			}
			mu.Lock()
			refs = append(refs, local...)
			mu.Unlock()
			return nil, nil
		}))
	}
	pool := jobpool.CreatePool(jobs, regionConcurrency)
	pool.Run()
	if pool.HasErrors() {
		return nil, pool.GetErrors()
	}
	return refs, nil
}

// fetchSpacesBucketDetails runs newSpacesBucket per ref in parallel.
// Results are written into an index-aligned slice so the final order
// is stable, then compacted to drop any nil entries — currently
// newSpacesBucket never returns (nil, nil), but the slot stays nil
// if a future variant ever filters a bucket without erroring, and we
// don't want nil leaking into the user-facing list.
func fetchSpacesBucketDetails(r *mqlDigitalocean, refs []bucketRef) ([]interface{}, error) {
	results := make([]interface{}, len(refs))
	jobs := make([]*jobpool.Job, 0, len(refs))
	for i, ref := range refs {
		idx, ref := i, ref
		jobs = append(jobs, jobpool.NewJob(func() (jobpool.JobResult, error) {
			res, err := newSpacesBucket(r, ref.client, ref.region, ref.b)
			if err != nil {
				return nil, err
			}
			results[idx] = res
			return nil, nil
		}))
	}
	pool := jobpool.CreatePool(jobs, bucketConcurrency)
	pool.Run()
	if pool.HasErrors() {
		return nil, pool.GetErrors()
	}
	out := make([]interface{}, 0, len(results))
	for _, res := range results {
		if res != nil {
			out = append(out, res)
		}
	}
	return out, nil
}

// newSpacesBucket builds a single bucket resource by fanning out
// to the access/encryption/versioning/policy/cors/lifecycle calls.
// Each call tolerates the "not configured" S3 error codes so an
// un-customized bucket still produces a resource with sensible nil
// defaults.
func newSpacesBucket(r *mqlDigitalocean, client *s3.Client, region string, b s3types.Bucket) (interface{}, error) {
	ctx := context.Background()
	name := aws.ToString(b.Name)

	var createdAt *time.Time
	if b.CreationDate != nil {
		t := *b.CreationDate
		createdAt = &t
	}

	publicAccessBlocked := false
	if pab, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: aws.String(name)}); err == nil {
		if cfg := pab.PublicAccessBlockConfiguration; cfg != nil {
			publicAccessBlocked = aws.ToBool(cfg.BlockPublicAcls) &&
				aws.ToBool(cfg.BlockPublicPolicy) &&
				aws.ToBool(cfg.IgnorePublicAcls) &&
				aws.ToBool(cfg.RestrictPublicBuckets)
		}
	} else if !isNoSuchConfiguration(err) {
		return nil, err
	}

	publicReadAcl, publicWriteAcl, authenticatedReadAcl := false, false, false
	aclGrants := []interface{}{}
	if acl, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{Bucket: aws.String(name)}); err == nil {
		for _, g := range acl.Grants {
			perm := string(g.Permission)
			grantee := g.Grantee
			var uri, ty, display string
			if grantee != nil {
				uri = aws.ToString(grantee.URI)
				ty = string(grantee.Type)
				display = aws.ToString(grantee.DisplayName)
			}
			aclGrants = append(aclGrants, map[string]interface{}{
				"granteeType":        ty,
				"granteeUri":         uri,
				"granteeDisplayName": display,
				"permission":         perm,
			})
			if uri == "http://acs.amazonaws.com/groups/global/AllUsers" {
				if perm == "READ" || perm == "FULL_CONTROL" {
					publicReadAcl = true
				}
				if perm == "WRITE" || perm == "FULL_CONTROL" {
					publicWriteAcl = true
				}
			}
			if uri == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" {
				if perm == "READ" || perm == "FULL_CONTROL" {
					authenticatedReadAcl = true
				}
			}
		}
	} else if isAccessDenied(err) {
		log.Warn().Err(err).Str("bucket", name).Msg("digitalocean> ACL access denied; bucket reported with no grants — audit results may be incomplete")
	} else {
		return nil, err
	}

	encryptionEnabled := false
	encryptionAlgorithm := ""
	encryptionKmsKeyId := ""
	if enc, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: aws.String(name)}); err == nil {
		if enc.ServerSideEncryptionConfiguration != nil && len(enc.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := enc.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				encryptionEnabled = true
				encryptionAlgorithm = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				encryptionKmsKeyId = aws.ToString(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID)
			}
		}
	} else if !isNoSuchConfiguration(err) {
		return nil, err
	}

	versioningStatus := ""
	mfaDeleteEnabled := false
	if v, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(name)}); err == nil {
		versioningStatus = string(v.Status)
		mfaDeleteEnabled = v.MFADelete == s3types.MFADeleteStatusEnabled
	} else if !isAccessDenied(err) {
		return nil, err
	}

	var policyDict interface{}
	if p, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: aws.String(name)}); err == nil {
		raw := aws.ToString(p.Policy)
		if raw != "" {
			var parsed interface{}
			if jerr := json.Unmarshal([]byte(raw), &parsed); jerr == nil {
				policyDict = parsed
			} else {
				policyDict = raw
			}
		}
	} else if !isNoSuchConfiguration(err) {
		return nil, err
	}

	corsRules := []interface{}{}
	if c, err := client.GetBucketCors(ctx, &s3.GetBucketCorsInput{Bucket: aws.String(name)}); err == nil {
		for _, rule := range c.CORSRules {
			corsRules = append(corsRules, map[string]interface{}{
				"id":             aws.ToString(rule.ID),
				"allowedHeaders": rule.AllowedHeaders,
				"allowedMethods": rule.AllowedMethods,
				"allowedOrigins": rule.AllowedOrigins,
				"exposeHeaders":  rule.ExposeHeaders,
				"maxAgeSeconds":  rule.MaxAgeSeconds,
			})
		}
	} else if !isNoSuchConfiguration(err) {
		return nil, err
	}

	lifecycleRules := []interface{}{}
	if l, err := client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{Bucket: aws.String(name)}); err == nil {
		for _, rule := range l.Rules {
			entry := map[string]interface{}{
				"id":     aws.ToString(rule.ID),
				"status": string(rule.Status),
			}
			if f := rule.Filter; f != nil {
				entry["prefix"] = aws.ToString(f.Prefix)
			}
			if rule.Expiration != nil {
				entry["expirationDays"] = rule.Expiration.Days
			}
			if rule.NoncurrentVersionExpiration != nil {
				entry["noncurrentVersionExpirationDays"] = rule.NoncurrentVersionExpiration.NoncurrentDays
			}
			if rule.AbortIncompleteMultipartUpload != nil {
				entry["abortIncompleteMultipartUploadDays"] = rule.AbortIncompleteMultipartUpload.DaysAfterInitiation
			}
			lifecycleRules = append(lifecycleRules, entry)
		}
	} else if !isNoSuchConfiguration(err) {
		return nil, err
	}

	return CreateResource(r.MqlRuntime, "digitalocean.spacesBucket", map[string]*llx.RawData{
		"name":                 llx.StringData(name),
		"region":               llx.StringData(region),
		"createdAt":            llx.TimeDataPtr(createdAt),
		"publicAccessBlocked":  llx.BoolData(publicAccessBlocked),
		"publicReadAcl":        llx.BoolData(publicReadAcl),
		"publicWriteAcl":       llx.BoolData(publicWriteAcl),
		"authenticatedReadAcl": llx.BoolData(authenticatedReadAcl),
		"aclGrants":            llx.ArrayData(aclGrants, types.Dict),
		"encryptionEnabled":    llx.BoolData(encryptionEnabled),
		"encryptionAlgorithm":  llx.StringData(encryptionAlgorithm),
		"encryptionKmsKeyId":   llx.StringData(encryptionKmsKeyId),
		"versioningStatus":     llx.StringData(versioningStatus),
		"mfaDeleteEnabled":     llx.BoolData(mfaDeleteEnabled),
		"policy":               llx.DictData(policyDict),
		"corsRules":            llx.ArrayData(corsRules, types.Dict),
		"lifecycleRules":       llx.ArrayData(lifecycleRules, types.Dict),
	})
}

// noSuchConfigurationCodes are the S3 error codes returned for
// well-formed buckets that have never been configured for a given
// property (no policy, no encryption, no CORS, etc.). Treat as
// soft-absent.
var noSuchConfigurationCodes = []string{
	"NoSuchBucketPolicy",
	"NoSuchPublicAccessBlockConfiguration",
	"ServerSideEncryptionConfigurationNotFoundError",
	"NoSuchCORSConfiguration",
	"NoSuchLifecycleConfiguration",
}

// isNoSuchConfiguration returns true when the S3 error code indicates
// the bucket has never been configured for that property. Both the
// typed APIError code and the raw error string are checked because
// Spaces sometimes surfaces the code only in the body text.
func isNoSuchConfiguration(err error) bool {
	if err == nil {
		return false
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		for _, code := range noSuchConfigurationCodes {
			if ae.ErrorCode() == code {
				return true
			}
		}
	}
	msg := err.Error()
	for _, code := range noSuchConfigurationCodes {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		return ae.ErrorCode() == "AccessDenied"
	}
	return strings.Contains(err.Error(), "AccessDenied")
}
