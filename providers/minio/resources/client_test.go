// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	madmin "github.com/minio/madmin-go/v4"
	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretKey = "test-secret-key"

// s3Fixture routes one bucket sub-resource request to a canned response.
type s3Fixture struct {
	status int
	body   string
}

// newS3Server replays real MinIO responses. Requests are keyed by the bucket
// and the sub-resource query parameter, which is how S3 addresses bucket
// configuration.
func newS3Server(t *testing.T, routes map[string]s3Fixture) *minio.Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := routeKey(r.URL)
		fixture, ok := routes[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(s3Error("NoSuchKey", "route "+key+" is not stubbed")))
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(fixture.status)
		_, _ = w.Write([]byte(fixture.body))
	}))
	t.Cleanup(srv.Close)

	endpoint := strings.TrimPrefix(srv.URL, "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4("minioadmin", testSecretKey, ""),
		Secure: false,
		Region: "us-east-1",
	})
	require.NoError(t, err)
	return client
}

func routeKey(u *url.URL) string {
	bucket := strings.Trim(u.Path, "/")
	for _, name := range []string{"versioning", "encryption", "object-lock", "lifecycle", "replication", "tagging", "location", "policy"} {
		if _, ok := u.Query()[name]; ok {
			return bucket + "?" + name
		}
	}
	if bucket == "" {
		return "/"
	}
	return bucket
}

func s3Error(code, message string) string {
	return `<?xml version="1.0" encoding="UTF-8"?><Error><Code>` + code +
		`</Code><Message>` + message + `</Message></Error>`
}

// TestReadBucketConfigurationFromRealResponses drives the S3 calls the bucket
// resource makes against the exact bodies a real deployment returned, so the
// field each accessor reads is pinned rather than assumed.
func TestReadBucketConfigurationFromRealResponses(t *testing.T) {
	ctx := context.Background()
	client := newS3Server(t, map[string]s3Fixture{
		"/":                        {200, fixture(t, "list_buckets.xml")},
		"private-data?location":    {200, fixture(t, "bucket_location.xml")},
		"private-data?versioning":  {200, fixture(t, "versioning_enabled.xml")},
		"private-data?encryption":  {200, fixture(t, "encryption_sse_s3.xml")},
		"private-data?object-lock": {200, fixture(t, "object_lock_governance.xml")},
		"private-data?lifecycle":   {200, fixture(t, "lifecycle.xml")},
		"private-data?tagging":     {200, fixture(t, "tagging.xml")},
		"kms-bucket?location":      {200, fixture(t, "bucket_location.xml")},
		"kms-bucket?encryption":    {200, fixture(t, "encryption_sse_kms.xml")},
		"plain?location":           {200, fixture(t, "bucket_location.xml")},
		"plain?versioning":         {200, fixture(t, "versioning_unset.xml")},
		"public-assets?location":   {200, fixture(t, "bucket_location.xml")},
		"public-assets?policy":     {200, fixture(t, "bucket_policy_anonymous_read.json")},
	})

	t.Run("bucket listing", func(t *testing.T) {
		buckets, err := client.ListBuckets(ctx)
		require.NoError(t, err)
		require.Len(t, buckets, 6, "every bucket the deployment reported is returned")

		names := make([]string, 0, len(buckets))
		for _, bucket := range buckets {
			names = append(names, bucket.Name)
			assert.False(t, bucket.CreationDate.IsZero(), "%s carries a creation date", bucket.Name)
		}
		assert.Equal(t, []string{
			"deny-only", "kms-bucket", "plain", "private-data", "public-assets", "wildcard-action",
		}, names)
	})

	t.Run("versioning", func(t *testing.T) {
		on, err := client.GetBucketVersioning(ctx, "private-data")
		require.NoError(t, err)
		assert.Equal(t, "Enabled", on.Status)

		// A bucket that never had versioning configured answers with an empty
		// status and no error, which must read as "off" rather than as unknown.
		off, err := client.GetBucketVersioning(ctx, "plain")
		require.NoError(t, err)
		assert.Equal(t, "", off.Status)
	})

	t.Run("encryption tells SSE-S3 from SSE-KMS", func(t *testing.T) {
		sses3, err := client.GetBucketEncryption(ctx, "private-data")
		require.NoError(t, err)
		require.Len(t, sses3.Rules, 1)
		assert.Equal(t, "AES256", sses3.Rules[0].Apply.SSEAlgorithm)
		assert.Equal(t, "", sses3.Rules[0].Apply.KmsMasterKeyID,
			"SSE-S3 names no key, so no key reference resolves")

		ssekms, err := client.GetBucketEncryption(ctx, "kms-bucket")
		require.NoError(t, err)
		require.Len(t, ssekms.Rules, 1)
		assert.Equal(t, "aws:kms", ssekms.Rules[0].Apply.SSEAlgorithm)
		assert.Equal(t, "minio-default-key", ssekms.Rules[0].Apply.KmsMasterKeyID)
	})

	t.Run("object lock", func(t *testing.T) {
		enabled, mode, validity, unit, err := client.GetObjectLockConfig(ctx, "private-data")
		require.NoError(t, err)
		assert.Equal(t, "Enabled", enabled)
		require.NotNil(t, mode)
		assert.Equal(t, minio.Governance, *mode)
		require.NotNil(t, validity)
		assert.Equal(t, uint(30), *validity)
		require.NotNil(t, unit)
		assert.Equal(t, minio.Days, *unit)
	})

	t.Run("lifecycle", func(t *testing.T) {
		config, err := client.GetBucketLifecycle(ctx, "private-data")
		require.NoError(t, err)
		require.Len(t, config.Rules, 1)

		rule := config.Rules[0]
		assert.Equal(t, "expire-old", rule.ID)
		assert.Equal(t, "Enabled", rule.Status)
		assert.Equal(t, 90, int(rule.Expiration.Days))
		assert.Equal(t, 30, int(rule.NoncurrentVersionExpiration.NoncurrentDays))

		// The prefix arrives inside Filter, not on the rule itself, which is
		// why the resource falls back to the filter's prefix.
		assert.Equal(t, "", rule.Prefix)
		assert.Equal(t, "logs/", rule.RuleFilter.Prefix)
	})

	t.Run("tagging", func(t *testing.T) {
		tagging, err := client.GetBucketTagging(ctx, "private-data")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod", "owner": "platform"}, tagging.ToMap())
	})

	t.Run("location", func(t *testing.T) {
		location, err := client.GetBucketLocation(ctx, "private-data")
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", location)
	})

	t.Run("policy", func(t *testing.T) {
		document, err := client.GetBucketPolicy(ctx, "public-assets")
		require.NoError(t, err)
		policy, err := parsePolicyDocument(document)
		require.NoError(t, err)
		assert.True(t, policyGrantsAnonymousAccess(policy))
	})
}

// TestAbsentConfigurationIsClassifiedAsAbsent replays the 404 bodies MinIO
// answers with when a bucket carries no configuration of a given kind.
func TestAbsentConfigurationIsClassifiedAsAbsent(t *testing.T) {
	ctx := context.Background()
	client := newS3Server(t, map[string]s3Fixture{
		"plain?location": {200, fixture(t, "bucket_location.xml")},
		"plain?encryption": {404, s3Error("ServerSideEncryptionConfigurationNotFoundError",
			"The server side encryption configuration was not found")},
		"plain?lifecycle": {404, s3Error("NoSuchLifecycleConfiguration",
			"The lifecycle configuration does not exist")},
		"plain?tagging": {404, s3Error("NoSuchTagSet", "The TagSet does not exist")},
		"plain?object-lock": {404, s3Error("ObjectLockConfigurationNotFoundError",
			"Object Lock configuration does not exist for this bucket")},
	})

	_, err := client.GetBucketEncryption(ctx, "plain")
	require.Error(t, err)
	assert.True(t, isS3ConfigAbsent(err), "no default encryption means unencrypted, not unknown")
	assert.False(t, isS3NoSuchBucket(err))

	_, err = client.GetBucketLifecycle(ctx, "plain")
	require.Error(t, err)
	assert.True(t, isS3ConfigAbsent(err))

	_, err = client.GetBucketTagging(ctx, "plain")
	require.Error(t, err)
	assert.True(t, isS3ConfigAbsent(err))

	_, _, _, _, err = client.GetObjectLockConfig(ctx, "plain")
	require.Error(t, err)
	assert.True(t, isS3ConfigAbsent(err))
}

// TestErrorsThatMustNotReadAsAbsent is the classifier's real job. Every case
// here would, if misclassified, turn a failure to read into a clean audit pass.
func TestErrorsThatMustNotReadAsAbsent(t *testing.T) {
	ctx := context.Background()

	t.Run("access denied", func(t *testing.T) {
		client := newS3Server(t, map[string]s3Fixture{
			"secret?location": {200, fixture(t, "bucket_location.xml")},
			"secret?encryption": {403, s3Error("AccessDenied",
				"Access Denied.")},
		})
		_, err := client.GetBucketEncryption(ctx, "secret")
		require.Error(t, err)
		assert.False(t, isS3ConfigAbsent(err),
			"a denied read says nothing about whether encryption is configured")
	})

	t.Run("bucket does not exist", func(t *testing.T) {
		client := newS3Server(t, map[string]s3Fixture{
			"gone?location":   {200, fixture(t, "bucket_location.xml")},
			"gone?encryption": {404, s3Error("NoSuchBucket", "The specified bucket does not exist")},
		})
		_, err := client.GetBucketEncryption(ctx, "gone")
		require.Error(t, err)
		assert.False(t, isS3ConfigAbsent(err), "a vanished bucket is not an unset setting")
		assert.True(t, isS3NoSuchBucket(err))
	})

	t.Run("server error", func(t *testing.T) {
		client := newS3Server(t, map[string]s3Fixture{
			"b?location":   {200, fixture(t, "bucket_location.xml")},
			"b?encryption": {500, s3Error("InternalError", "We encountered an internal error.")},
		})
		_, err := client.GetBucketEncryption(ctx, "b")
		require.Error(t, err)
		assert.False(t, isS3ConfigAbsent(err))
	})

	t.Run("transport failure", func(t *testing.T) {
		// A connection that never reached a server produces no S3 error at all.
		// This is the case that matters most: a network blip that read as
		// "not configured" would pass an encryption audit on a bucket nobody
		// managed to look at.
		client, err := minio.New("127.0.0.1:1", &minio.Options{
			Creds: miniocreds.NewStaticV4("k", "s", ""), Secure: false, Region: "us-east-1",
		})
		require.NoError(t, err)

		_, err = client.GetBucketEncryption(ctx, "b")
		require.Error(t, err)
		assert.False(t, isS3ConfigAbsent(err))
		assert.False(t, isS3NoSuchBucket(err))
		assert.False(t, isAdminNotFound(err))
	})

	t.Run("nil error is never absent", func(t *testing.T) {
		assert.False(t, isS3ConfigAbsent(nil))
		assert.False(t, isS3NoSuchBucket(nil))
		assert.False(t, isAdminNotFound(nil))
	})
}

// ------------------------------------------------------------ admin API

// newAdminServer replays admin API responses. Bodies for the endpoints MinIO
// encrypts are encrypted with the same scheme the deployment uses, so the
// client's own decryption is exercised rather than bypassed.
func newAdminServer(t *testing.T, routes map[string]s3Fixture, encrypted map[string]bool) *madmin.AdminClient {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The deployment this was captured from answers 426 on the v4 prefix
		// and serves v3, which is the fallback madmin performs. Serving both
		// keeps the test independent of which prefix the client tries first.
		path := strings.TrimPrefix(r.URL.Path, "/minio/admin/v3")
		path = strings.TrimPrefix(path, "/minio/admin/v4")

		fixture, ok := routes[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Code":"XMinioAdminNoSuchRoute","Message":"` + path + `"}`))
			return
		}
		body := []byte(fixture.body)
		if encrypted[path] {
			enc, err := madmin.EncryptData(testSecretKey, body)
			require.NoError(t, err)
			body = enc
		}
		w.WriteHeader(fixture.status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	client, err := madmin.NewWithOptions(strings.TrimPrefix(srv.URL, "http://"), &madmin.Options{
		Creds:  miniocreds.NewStaticV4("minioadmin", testSecretKey, ""),
		Secure: false,
	})
	require.NoError(t, err)
	return client
}

func TestAdminListingsFromRealResponses(t *testing.T) {
	ctx := context.Background()
	client := newAdminServer(t,
		map[string]s3Fixture{
			"/list-users":           {200, fixture(t, "list_users.json")},
			"/groups":               {200, `["devs"]`},
			"/group":                {200, fixture(t, "group_devs.json")},
			"/list-canned-policies": {200, fixture(t, "list_canned_policies.json")},
			"/info":                 {200, fixture(t, "server_info.json")},
			"/get-config-kv":        {200, fixture(t, "config_kv_audit_webhook.txt")},
			"/get-bucket-quota":     {200, fixture(t, "bucket_quota.json")},
		},
		map[string]bool{
			"/list-users":    true,
			"/get-config-kv": true,
		},
	)

	t.Run("users", func(t *testing.T) {
		users, err := client.ListUsers(ctx)
		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, madmin.AccountDisabled, users["bob"].Status)
		assert.True(t, users["bob"].UpdatedAt.IsZero())
	})

	t.Run("groups", func(t *testing.T) {
		groups, err := client.ListGroups(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"devs"}, groups)

		desc, err := client.GetGroupDescription(ctx, "devs")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice"}, desc.Members)
	})

	t.Run("policies", func(t *testing.T) {
		policies, err := client.ListCannedPolicies(ctx)
		require.NoError(t, err)
		assert.Len(t, policies, 7)
		assert.Contains(t, policies, "consoleAdmin")
	})

	t.Run("server info", func(t *testing.T) {
		info, err := client.ServerInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "us-east-1", info.Region)
		require.Len(t, info.Servers, 1)
	})

	t.Run("config", func(t *testing.T) {
		raw, err := client.GetConfigKV(ctx, "audit_webhook")
		require.NoError(t, err)
		configs, err := madmin.ParseServerConfigOutput(string(raw))
		require.NoError(t, err)
		targets := webhookTargetsFromConfig(webhookTypeAudit, configs, nil)
		assert.Len(t, targets, 4)
	})

	t.Run("quota", func(t *testing.T) {
		quota, err := client.GetBucketQuota(ctx, "private-data")
		require.NoError(t, err)
		assert.Equal(t, uint64(1073741824), quota.Size)
	})
}

func TestAdminNotFoundClassifier(t *testing.T) {
	ctx := context.Background()
	client := newAdminServer(t, map[string]s3Fixture{
		"/info-canned-policy": {400, `{"Code":"XMinioAdminNoSuchPolicy","Message":"policy does not exist"}`},
		"/group":              {500, `{"Code":"InternalError","Message":"boom"}`},
	}, nil)

	_, err := client.InfoCannedPolicy(ctx, "nope")
	require.Error(t, err)
	assert.True(t, isAdminNotFound(err))

	_, err = client.GetGroupDescription(ctx, "devs")
	require.Error(t, err)
	assert.False(t, isAdminNotFound(err), "a server error is not an absent group")

	unreachable, err := madmin.NewWithOptions("127.0.0.1:1", &madmin.Options{
		Creds: miniocreds.NewStaticV4("k", "s", ""), Secure: false,
	})
	require.NoError(t, err)
	_, err = unreachable.ListUsers(ctx)
	require.Error(t, err)
	assert.False(t, isAdminNotFound(err), "a transport failure is not an absent object")
}
