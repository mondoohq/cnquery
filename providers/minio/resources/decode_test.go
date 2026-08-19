// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	madmin "github.com/minio/madmin-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
)

// TestDecodeRealUserListing pins the user listing against the payload a real
// deployment returned. bob is the case that matters: he was created and then
// disabled, and the deployment reports his updatedAt as the zero time.
func TestDecodeRealUserListing(t *testing.T) {
	var users map[string]madmin.UserInfo
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "list_users.json")), &users))
	require.Len(t, users, 2)

	alice := users["alice"]
	assert.Equal(t, "scoped-read", alice.PolicyName)
	assert.Equal(t, madmin.AccountEnabled, alice.Status)
	assert.Equal(t, []string{"devs"}, alice.MemberOf)
	assert.False(t, alice.UpdatedAt.IsZero())
	assert.Empty(t, alice.SecretKey, "the listing carried no secret key")

	bob := users["bob"]
	assert.Equal(t, madmin.AccountDisabled, bob.Status)
	assert.Empty(t, bob.PolicyName, "bob has no policy attached")
	assert.True(t, bob.UpdatedAt.IsZero(), "the deployment reported 0001-01-01 for bob")

	// The zero time must reach the schema as null. Reporting 1 January year 1
	// as a real date makes every "changed since" comparison silently true.
	assert.Equal(t, llx.NilData, userSchemaArgs("bob", bob)["updatedAt"])
	assert.NotEqual(t, llx.NilData, userSchemaArgs("alice", alice)["updatedAt"])
}

func TestUserSchemaArgs(t *testing.T) {
	args := userSchemaArgs("alice", madmin.UserInfo{
		Status:    madmin.AccountEnabled,
		UpdatedAt: time.Date(2026, 8, 19, 13, 44, 53, 0, time.UTC),
	})
	assert.Equal(t, "user/alice", args["__id"].Value)
	assert.Equal(t, "alice", args["name"].Value)
	assert.Equal(t, "enabled", args["status"].Value)
	assert.Equal(t, true, args["enabled"].Value)

	disabled := userSchemaArgs("bob", madmin.UserInfo{Status: madmin.AccountDisabled})
	assert.Equal(t, false, disabled["enabled"].Value)

	// A status the deployment never set must not read as enabled.
	unknown := userSchemaArgs("carol", madmin.UserInfo{})
	assert.Equal(t, false, unknown["enabled"].Value)
	assert.Equal(t, "", unknown["status"].Value)
}

func TestDecodeRealGroupDescription(t *testing.T) {
	var desc madmin.GroupDesc
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "group_devs.json")), &desc))

	assert.Equal(t, "devs", desc.Name)
	assert.Equal(t, "enabled", desc.Status)
	assert.Equal(t, []string{"alice"}, desc.Members)
	assert.Equal(t, "custom-wildcard", desc.Policy)

	args := groupSchemaArgs(desc)
	assert.Equal(t, "group/devs", args["__id"].Value)
	assert.Equal(t, true, args["enabled"].Value)
	assert.NotEqual(t, llx.NilData, args["updatedAt"])

	// A group the deployment never stamped reports null, not year 1.
	assert.Equal(t, llx.NilData, groupSchemaArgs(madmin.GroupDesc{Name: "x"})["updatedAt"])
}

func TestDecodeRealServiceAccountListing(t *testing.T) {
	var listed madmin.ListServiceAccountsResp
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "list_service_accounts.json")), &listed))
	require.Len(t, listed.Accounts, 1)

	account := listed.Accounts[0]
	assert.Equal(t, "alicesvcacct0001", account.AccessKey)
	assert.Equal(t, "alice", account.ParentUser)
	assert.Equal(t, "on", account.AccountStatus)
	assert.True(t, account.ImpliedPolicy)
	assert.Equal(t, "ci-pipeline", account.Name)
	assert.Equal(t, "used by CI", account.Description)
	require.NotNil(t, account.Expiration)

	args := serviceAccountSchemaArgs(account)
	assert.Equal(t, "serviceAccount/alicesvcacct0001", args["__id"].Value)
	assert.Equal(t, true, args["impliedPolicy"].Value)
	assert.NotEqual(t, llx.NilData, args["expiresAt"])

	// An account that never expires reports null. A zero time here would read
	// as an expiry in the year 1, so every "is it expired" check would pass.
	never := serviceAccountSchemaArgs(madmin.ServiceAccountInfo{AccessKey: "k"})
	assert.Equal(t, llx.NilData, never["expiresAt"])

	// So must an expiration explicitly stamped as the zero time.
	zero := time.Time{}
	zeroStamped := serviceAccountSchemaArgs(madmin.ServiceAccountInfo{AccessKey: "k", Expiration: &zero})
	assert.Equal(t, llx.NilData, zeroStamped["expiresAt"])
}

func TestDecodeRealServiceAccountInfo(t *testing.T) {
	var info madmin.InfoServiceAccountResp
	raw := fixture(t, "info_service_account.json")
	require.NoError(t, json.Unmarshal([]byte(raw), &info))

	assert.Equal(t, "alice", info.ParentUser)
	assert.True(t, info.ImpliedPolicy)
	require.NotEmpty(t, info.Policy)

	// The policy in force is the union of the account's own grants and the
	// parent user's, which is why an implied-policy account still reports one.
	policy, err := parsePolicyDocument(info.Policy)
	require.NoError(t, err)
	require.Len(t, policy.Statements, 2)
	assert.True(t, policyHasWildcardAction(policy))
	assert.True(t, policyHasWildcardResource(policy))
	assert.False(t, policyGrantsAdminAccess(policy))
}

func TestDecodeRealPolicyInfo(t *testing.T) {
	var info madmin.PolicyInfo
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "info_canned_policy.json")), &info))

	assert.Equal(t, "custom-wildcard", info.PolicyName)
	assert.False(t, info.CreateDate.IsZero())
	assert.False(t, info.UpdateDate.IsZero())

	policy, err := parsePolicyDocument(string(info.Policy))
	require.NoError(t, err)
	assert.True(t, policyHasWildcardAction(policy))
}

func TestDecodeRealBucketQuota(t *testing.T) {
	var quota madmin.BucketQuota
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "bucket_quota.json")), &quota))

	// The payload carries both a legacy "quota" of 0 and the "size" that
	// actually holds the limit. Reading the wrong one reports an unlimited
	// bucket on a bucket with a 1 GiB cap.
	assert.Equal(t, uint64(1073741824), quota.Size)
	assert.Equal(t, madmin.HardQuota, quota.Type)

	var none madmin.BucketQuota
	require.NoError(t, json.Unmarshal([]byte(`{"quota":0,"size":0,"rate":0,"requests":0}`), &none))
	assert.Equal(t, uint64(0), none.Size)
	assert.Equal(t, madmin.QuotaType(""), none.Type, "an unset quota reports no type")
}

func TestDecodeRealKmsKeyStatus(t *testing.T) {
	t.Run("healthy key", func(t *testing.T) {
		var status madmin.KMSKeyStatus
		require.NoError(t, json.Unmarshal([]byte(fixture(t, "kms_key_status_ok.json")), &status))
		assert.Equal(t, "minio-default-key", status.KeyID)
		assert.Empty(t, status.EncryptionErr)
		assert.Empty(t, status.DecryptionErr)
	})

	t.Run("key that does not exist", func(t *testing.T) {
		var status madmin.KMSKeyStatus
		require.NoError(t, json.Unmarshal([]byte(fixture(t, "kms_key_status_missing.json")), &status))
		// The request succeeded. The failure is reported in the body, which is
		// why health cannot be derived from the request having worked.
		assert.Equal(t, "nonexistent-key", status.KeyID)
		assert.Equal(t, "key with given key ID does not exist", status.EncryptionErr)
	})
}

func TestDecodeRealServerInfo(t *testing.T) {
	var info madmin.InfoMessage
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "server_info.json")), &info))

	assert.Equal(t, "us-east-1", info.Region)
	assert.Equal(t, "d913d08f-d114-4eb0-9922-6a2a117b7742", info.DeploymentID)
	assert.Equal(t, madmin.ErasureType, info.Backend.Type)
	assert.Equal(t, 4, info.Backend.OnlineDisks)
	assert.Equal(t, 0, info.Backend.OfflineDisks)
	require.Len(t, info.Servers, 1)

	server := info.Servers[0]
	assert.Equal(t, "127.0.0.1:9000", server.Endpoint)
	assert.Equal(t, "online", server.State)
	total, online := driveCounts(server.Disks)
	assert.Equal(t, int64(4), total)
	assert.Equal(t, int64(4), online)

	// "mode" is the health of the deployment, not its topology: it reads
	// "online" on a single-server deployment. The topology has to come from
	// the server count, or every deployment would report the same mode.
	assert.Equal(t, "online", info.Mode)

	// The environment listing is redacted by the deployment before it leaves.
	require.Contains(t, server.MinioEnvVars, "MINIO_ROOT_PASSWORD")
	assert.Equal(t, "*** EXISTS, REDACTED ***", server.MinioEnvVars["MINIO_ROOT_PASSWORD"])
}

func TestDriveCounts(t *testing.T) {
	total, online := driveCounts(nil)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, int64(0), online)

	total, online = driveCounts([]madmin.Disk{
		{State: "ok"}, {State: "offline"}, {State: "ok"}, {State: ""},
	})
	assert.Equal(t, int64(4), total)
	assert.Equal(t, int64(2), online, "only a drive reporting ok counts as online")
}

func TestSplitCommaList(t *testing.T) {
	assert.Equal(t, []any{"*"}, splitCommaList("*"))
	assert.Equal(t, []any{}, splitCommaList(""))
	assert.Equal(t, []any{"https://a.example", "https://b.example"},
		splitCommaList("https://a.example, https://b.example"))
	assert.Equal(t, []any{}, splitCommaList(" , "))
}

func TestParseOnOff(t *testing.T) {
	for _, v := range []string{"on", "ON", "true", "yes", "enabled", " on "} {
		assert.True(t, parseOnOff(v), "%q", v)
	}
	for _, v := range []string{"off", "", "false", "no", "0", "1", "nonsense"} {
		assert.False(t, parseOnOff(v), "%q", v)
	}
}

func TestParseInt(t *testing.T) {
	assert.Equal(t, int64(100000), parseInt("100000"))
	assert.Equal(t, int64(0), parseInt(""))
	assert.Equal(t, int64(0), parseInt("not a number"))
	assert.Equal(t, int64(5), parseInt(" 5 "))
}
