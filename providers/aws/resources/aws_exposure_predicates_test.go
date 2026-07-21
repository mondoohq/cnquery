// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountIdFromPrincipal(t *testing.T) {
	assert.Equal(t, "123456789012", accountIdFromPrincipal("123456789012"))
	assert.Equal(t, "123456789012", accountIdFromPrincipal("arn:aws:iam::123456789012:root"))
	assert.Equal(t, "123456789012", accountIdFromPrincipal("arn:aws:iam::123456789012:role/Example"))
	assert.Equal(t, "", accountIdFromPrincipal("*"))
	assert.Equal(t, "", accountIdFromPrincipal("ec2.amazonaws.com"))
	assert.Equal(t, "", accountIdFromPrincipal(""))
	assert.Equal(t, "", accountIdFromPrincipal("arn:aws:iam::aws:policy/Foo")) // "aws" is not an account id
}

func TestIsAwsAccountId(t *testing.T) {
	assert.True(t, isAwsAccountId("123456789012"))
	assert.False(t, isAwsAccountId("12345678901"))   // 11 digits
	assert.False(t, isAwsAccountId("1234567890123")) // 13 digits
	assert.False(t, isAwsAccountId("12345678901a"))
	assert.False(t, isAwsAccountId(""))
}

func TestLooksLikeSecretKey(t *testing.T) {
	for _, k := range []string{"DB_PASSWORD", "API_TOKEN", "MY_SECRET", "aws_access_key_id", "ApiKey", "PRIVATE_KEY", "passwd"} {
		assert.True(t, looksLikeSecretKey(k), k)
	}
	for _, k := range []string{"KMS_KEY_ID", "REGION", "LOG_LEVEL", "BUCKET_NAME", "KEY_NAME"} {
		assert.False(t, looksLikeSecretKey(k), k)
	}
}
