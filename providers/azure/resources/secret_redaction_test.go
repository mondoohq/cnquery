// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/util/convert"
)

// These tests assert on the SERIALIZED form, not on the struct fields, because
// serialization is what actually reaches MQL. A redaction that nils a field the
// marshaller does not emit would pass a struct-level assertion and still leak.

const (
	fakeStorageConnString = "DefaultEndpointsProtocol=https;AccountName=acct;AccountKey=SUPERSECRETKEY==;"
	fakeClientSecret      = "SUPERSECRETCLIENTSECRET"
)

func TestRedactedRedisConfigurationDropsStorageCredentials(t *testing.T) {
	cfg := &armredis.CommonPropertiesRedisConfiguration{
		AofStorageConnectionString0: to.Ptr(fakeStorageConnString + "aof0"),
		AofStorageConnectionString1: to.Ptr(fakeStorageConnString + "aof1"),
		RdbStorageConnectionString:  to.Ptr(fakeStorageConnString + "rdb"),
		RdbBackupEnabled:            to.Ptr("true"),
		MaxmemoryPolicy:             to.Ptr("allkeys-lru"),
	}

	dict, err := convert.JsonToDict(redactedRedisConfiguration(cfg))
	require.NoError(t, err)
	serialized := mustJSON(t, dict)

	assert.NotContains(t, serialized, "SUPERSECRETKEY",
		"the storage account key must not reach MQL")
	assert.NotContains(t, serialized, "AccountKey")

	// The audit signal has to survive the redaction, or the fix trades one
	// wrong answer for another.
	assert.Contains(t, serialized, "true", "rdbBackupEnabled is still reported")
	assert.Contains(t, serialized, "allkeys-lru")

	// The source is not mutated -- callers downstream still see the original.
	assert.Equal(t, fakeStorageConnString+"rdb", *cfg.RdbStorageConnectionString)

	assert.Nil(t, redactedRedisConfiguration(nil))
}

// TestRedactedRedisResourceDropsStorageCredentials covers the second, easier to
// miss leak: properties serializes the whole ResourceInfo, so redacting only
// the redisConfiguration field would have left the same secrets readable one
// field over.
func TestRedactedRedisResourceDropsStorageCredentials(t *testing.T) {
	cache := &armredis.ResourceInfo{
		Name: to.Ptr("my-cache"),
		Properties: &armredis.Properties{
			RedisConfiguration: &armredis.CommonPropertiesRedisConfiguration{
				RdbStorageConnectionString: to.Ptr(fakeStorageConnString),
			},
		},
	}

	dict, err := convert.JsonToDict(redactedRedisResource(cache))
	require.NoError(t, err)
	serialized := mustJSON(t, dict)

	assert.NotContains(t, serialized, "SUPERSECRETKEY")
	assert.Contains(t, serialized, "my-cache", "the rest of the record is untouched")

	// The copy must be deep enough that the caller's struct is unchanged; the
	// same *ResourceInfo is used afterwards to seed the resource's caches.
	require.NotNil(t, cache.Properties.RedisConfiguration.RdbStorageConnectionString)
	assert.Equal(t, fakeStorageConnString, *cache.Properties.RedisConfiguration.RdbStorageConnectionString)

	assert.Nil(t, redactedRedisResource(nil))

	// A cache with no properties must not panic.
	assert.NotNil(t, redactedRedisResource(&armredis.ResourceInfo{Name: to.Ptr("bare")}))
}

func TestRedactedAuthSettingsDropsEveryProviderSecret(t *testing.T) {
	props := &web.SiteAuthSettingsProperties{
		Enabled:                      to.Ptr(true),
		ClientID:                     to.Ptr("client-id-is-not-a-secret"),
		ClientSecret:                 to.Ptr(fakeClientSecret + "-aad"),
		FacebookAppSecret:            to.Ptr(fakeClientSecret + "-fb"),
		GitHubClientSecret:           to.Ptr(fakeClientSecret + "-gh"),
		GoogleClientSecret:           to.Ptr(fakeClientSecret + "-goog"),
		MicrosoftAccountClientSecret: to.Ptr(fakeClientSecret + "-msa"),
		TwitterConsumerSecret:        to.Ptr(fakeClientSecret + "-tw"),
	}

	dict, err := convert.JsonToDict(redactedAuthSettings(props))
	require.NoError(t, err)
	serialized := mustJSON(t, dict)

	// One assertion per provider: a redaction that misses one is the whole bug.
	for _, suffix := range []string{"-aad", "-fb", "-gh", "-goog", "-msa", "-tw"} {
		assert.NotContains(t, serialized, fakeClientSecret+suffix,
			"secret %q must not reach MQL", suffix)
	}
	assert.NotContains(t, serialized, fakeClientSecret)

	// Everything that is not a secret is still reported: the resource exists to
	// say whether Easy Auth is on and how it is configured.
	assert.Contains(t, serialized, "client-id-is-not-a-secret")
	assert.Contains(t, serialized, "true")

	assert.Equal(t, fakeClientSecret+"-aad", *props.ClientSecret, "source not mutated")
	assert.Nil(t, redactedAuthSettings(nil))
}

// mustJSON serializes exactly as the value will reach MQL. It deliberately does
// NOT normalize case: the assertions below search for upper-case secret
// markers, and lower-casing here would make every NotContains unfalsifiable.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
