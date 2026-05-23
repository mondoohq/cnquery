// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cf_types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDriftStatus(t *testing.T) {
	// Empty maps to NOT_CHECKED because the SDK enum's zero value carries
	// no semantic meaning, but the field's contract promises one of four
	// documented strings.
	cases := []struct {
		in, want string
	}{
		{"", "NOT_CHECKED"},
		{"NOT_CHECKED", "NOT_CHECKED"},
		{"DRIFTED", "DRIFTED"},
		{"IN_SYNC", "IN_SYNC"},
		{"UNKNOWN", "UNKNOWN"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeDriftStatus(tc.in))
		})
	}
}

func TestCfnTagsToMap(t *testing.T) {
	t.Run("empty input returns empty map (not nil)", func(t *testing.T) {
		got := cfnTagsToMap(nil)
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("nil key entries are skipped", func(t *testing.T) {
		// Nil keys can't be indexed and would panic on deref. Defensive
		// skip — AWS shouldn't return them but we don't trust the wire.
		got := cfnTagsToMap([]cf_types.Tag{
			{Key: nil, Value: aws.String("v")},
			{Key: aws.String("ok"), Value: aws.String("yes")},
		})
		assert.Equal(t, map[string]any{"ok": "yes"}, got)
	})

	t.Run("nil value becomes empty string", func(t *testing.T) {
		got := cfnTagsToMap([]cf_types.Tag{
			{Key: aws.String("k"), Value: nil},
		})
		assert.Equal(t, map[string]any{"k": ""}, got)
	})
}

func TestStackParameterToDict(t *testing.T) {
	t.Run("required fields only emits key/value", func(t *testing.T) {
		got := stackParameterToDict(cf_types.Parameter{
			ParameterKey:   aws.String("InstanceType"),
			ParameterValue: aws.String("t3.micro"),
		})
		assert.Equal(t, map[string]any{
			"key":   "InstanceType",
			"value": "t3.micro",
		}, got)
	})

	t.Run("uses lowercase mql-style keys, not SDK CamelCase", func(t *testing.T) {
		// Regression: prior implementation round-tripped through
		// json.Marshal on cf_types.Parameter, leaking SDK field names
		// (ParameterKey/ParameterValue) to mql users.
		got := stackParameterToDict(cf_types.Parameter{
			ParameterKey:   aws.String("k"),
			ParameterValue: aws.String("v"),
		})
		_, hasCamel := got["ParameterKey"]
		assert.False(t, hasCamel, "must not surface SDK CamelCase")
		assert.Contains(t, got, "key")
		assert.Contains(t, got, "value")
	})

	t.Run("ResolvedValue only emitted when set", func(t *testing.T) {
		without := stackParameterToDict(cf_types.Parameter{
			ParameterKey:   aws.String("k"),
			ParameterValue: aws.String("v"),
		})
		assert.NotContains(t, without, "resolvedValue")

		with := stackParameterToDict(cf_types.Parameter{
			ParameterKey:   aws.String("k"),
			ParameterValue: aws.String("v"),
			ResolvedValue:  aws.String("ssm-resolved"),
		})
		assert.Equal(t, "ssm-resolved", with["resolvedValue"])
	})

	t.Run("UsePreviousValue only emitted when set", func(t *testing.T) {
		got := stackParameterToDict(cf_types.Parameter{
			ParameterKey:     aws.String("k"),
			ParameterValue:   aws.String("v"),
			UsePreviousValue: aws.Bool(true),
		})
		assert.Equal(t, true, got["usePreviousValue"])
	})

	t.Run("nil key/value collapse to empty strings", func(t *testing.T) {
		got := stackParameterToDict(cf_types.Parameter{})
		assert.Equal(t, map[string]any{"key": "", "value": ""}, got)
	})
}

func TestStackOutputToDict(t *testing.T) {
	t.Run("required fields only emits key/value", func(t *testing.T) {
		got := stackOutputToDict(cf_types.Output{
			OutputKey:   aws.String("BucketName"),
			OutputValue: aws.String("my-bucket"),
		})
		assert.Equal(t, map[string]any{
			"key":   "BucketName",
			"value": "my-bucket",
		}, got)
	})

	t.Run("description and exportName emitted only when set", func(t *testing.T) {
		full := stackOutputToDict(cf_types.Output{
			OutputKey:   aws.String("k"),
			OutputValue: aws.String("v"),
			Description: aws.String("desc"),
			ExportName:  aws.String("export"),
		})
		assert.Equal(t, "desc", full["description"])
		assert.Equal(t, "export", full["exportName"])

		minimal := stackOutputToDict(cf_types.Output{
			OutputKey:   aws.String("k"),
			OutputValue: aws.String("v"),
		})
		assert.NotContains(t, minimal, "description")
		assert.NotContains(t, minimal, "exportName")
	})
}
