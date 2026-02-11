// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLambdaArn(t *testing.T) {
	t.Run("standard function ARN", func(t *testing.T) {
		arn := getLambdaArn("my-function", "us-east-1", "123456789012")
		assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:my-function", arn)
	})

	t.Run("different region and account", func(t *testing.T) {
		arn := getLambdaArn("process-orders", "eu-west-1", "987654321098")
		assert.Equal(t, "arn:aws:lambda:eu-west-1:987654321098:function:process-orders", arn)
	})

	t.Run("empty account ID", func(t *testing.T) {
		arn := getLambdaArn("my-function", "us-east-1", "")
		assert.Equal(t, "arn:aws:lambda:us-east-1::function:my-function", arn)
	})
}

func TestLambdaFunctionRole(t *testing.T) {
	t.Run("nil cacheRoleArn sets null state", func(t *testing.T) {
		fn := &mqlAwsLambdaFunction{}
		// cacheRoleArn is nil by default
		result, err := fn.role()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fn.Role.IsNull())
		assert.True(t, fn.Role.IsSet())
	})

	t.Run("empty cacheRoleArn sets null state", func(t *testing.T) {
		fn := &mqlAwsLambdaFunction{}
		empty := ""
		fn.cacheRoleArn = &empty
		result, err := fn.role()
		require.NoError(t, err)
		require.Nil(t, result)
		assert.True(t, fn.Role.IsNull())
		assert.True(t, fn.Role.IsSet())
	})
}
