// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "", firstNonEmpty())
	assert.Equal(t, "", firstNonEmpty("", ""))
	assert.Equal(t, "a", firstNonEmpty("a", "b"))
	assert.Equal(t, "b", firstNonEmpty("", "b", "c"))
	assert.Equal(t, "c", firstNonEmpty("", "", "c"))
}

func TestEndpoint(t *testing.T) {
	t.Run("RAM is a global service", func(t *testing.T) {
		assert.Equal(t, "ram.aliyuncs.com", endpoint("ram", "cn-hangzhou"))
		assert.Equal(t, "ram.aliyuncs.com", endpoint("ram", "us-west-1"))
	})

	t.Run("regional services are region-scoped", func(t *testing.T) {
		assert.Equal(t, "ecs.cn-hangzhou.aliyuncs.com", endpoint("ecs", "cn-hangzhou"))
		assert.Equal(t, "vpc.ap-southeast-1.aliyuncs.com", endpoint("vpc", "ap-southeast-1"))
		assert.Equal(t, "r-kvstore.cn-beijing.aliyuncs.com", endpoint("r-kvstore", "cn-beijing"))
		assert.Equal(t, "mongodb.cn-hangzhou.aliyuncs.com", endpoint("mongodb", "cn-hangzhou"))
		assert.Equal(t, "polardb.us-east-1.aliyuncs.com", endpoint("polardb", "us-east-1"))
	})
}

func TestResolveCredential(t *testing.T) {
	t.Run("access key pair selects the access_key credential", func(t *testing.T) {
		cred, err := resolveCredential(map[string]string{}, "id", "secret", "")
		require.NoError(t, err)
		require.NotNil(t, cred)
		require.NotNil(t, cred.GetType())
		assert.Equal(t, "access_key", *cred.GetType())
	})

	t.Run("access key with STS token selects the sts credential", func(t *testing.T) {
		cred, err := resolveCredential(map[string]string{}, "id", "secret", "token")
		require.NoError(t, err)
		require.NotNil(t, cred)
		require.NotNil(t, cred.GetType())
		assert.Equal(t, "sts", *cred.GetType())
	})

	t.Run("role ARN with an access key selects the ram_role_arn credential", func(t *testing.T) {
		cred, err := resolveCredential(
			map[string]string{OptionRoleArn: "acs:ram::123456789012:role/mondoo"},
			"id", "secret", "",
		)
		require.NoError(t, err)
		require.NotNil(t, cred)
		require.NotNil(t, cred.GetType())
		assert.Equal(t, "ram_role_arn", *cred.GetType())
	})

	t.Run("role ARN without an access key does not select ram_role_arn", func(t *testing.T) {
		// With no static access key the role cannot be assumed directly, so the
		// resolver must fall through to the default credential chain rather than
		// building a ram_role_arn credential.
		cred, err := resolveCredential(
			map[string]string{OptionRoleArn: "acs:ram::123456789012:role/mondoo"},
			"", "", "",
		)
		if err == nil && cred != nil && cred.GetType() != nil {
			assert.NotEqual(t, "ram_role_arn", *cred.GetType())
		}
	})
}
