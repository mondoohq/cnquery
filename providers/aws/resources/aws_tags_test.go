// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	emrtypes "github.com/aws/aws-sdk-go-v2/service/emr/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyValueTag mirrors the shape of the vast majority of AWS SDK tag structs
// (Key/Value, both *string).
type keyValueTag struct {
	Key   *string
	Value *string
}

// tagKeyValueTag mirrors the KMS-style shape (TagKey/TagValue) to prove the
// generic helper handles alternate field names via its accessors.
type tagKeyValueTag struct {
	TagKey   *string
	TagValue *string
}

func kvKey(t keyValueTag) *string   { return t.Key }
func kvValue(t keyValueTag) *string { return t.Value }

func TestTagsToMap(t *testing.T) {
	t.Run("nil slice returns non-nil empty map", func(t *testing.T) {
		got := tagsToMap(nil, kvKey, kvValue)
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("empty slice returns non-nil empty map", func(t *testing.T) {
		got := tagsToMap([]keyValueTag{}, kvKey, kvValue)
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("maps key/value pairs", func(t *testing.T) {
		got := tagsToMap([]keyValueTag{
			{Key: aws.String("Name"), Value: aws.String("web")},
			{Key: aws.String("env"), Value: aws.String("prod")},
		}, kvKey, kvValue)
		assert.Equal(t, map[string]any{"Name": "web", "env": "prod"}, got)
	})

	t.Run("skips entries with a nil key", func(t *testing.T) {
		got := tagsToMap([]keyValueTag{
			{Key: nil, Value: aws.String("orphan")},
			{Key: aws.String("ok"), Value: aws.String("yes")},
		}, kvKey, kvValue)
		assert.Equal(t, map[string]any{"ok": "yes"}, got)
	})

	t.Run("coerces a nil value to empty string, keeping the key", func(t *testing.T) {
		got := tagsToMap([]keyValueTag{
			{Key: aws.String("present-empty"), Value: nil},
		}, kvKey, kvValue)
		assert.Equal(t, map[string]any{"present-empty": ""}, got)
	})

	t.Run("values are stored as strings", func(t *testing.T) {
		got := tagsToMap([]keyValueTag{
			{Key: aws.String("k"), Value: aws.String("v")},
		}, kvKey, kvValue)
		v, ok := got["k"].(string)
		require.True(t, ok, "value must be a string, got %T", got["k"])
		assert.Equal(t, "v", v)
	})

	t.Run("works with alternate field names (TagKey/TagValue)", func(t *testing.T) {
		got := tagsToMap([]tagKeyValueTag{
			{TagKey: aws.String("k"), TagValue: aws.String("v")},
			{TagKey: nil, TagValue: aws.String("dropped")},
		},
			func(t tagKeyValueTag) *string { return t.TagKey },
			func(t tagKeyValueTag) *string { return t.TagValue },
		)
		assert.Equal(t, map[string]any{"k": "v"}, got)
	})
}

// Wrapper spot-checks: guard the mechanical rewrite of the per-service
// converters against a Key/Value accessor swap or wrong output type. The
// common Tag shape is already covered by TestEc2TagsToMap (Tag -> string) and
// TestCfnTagsToMap (Tag -> any); these cover the structurally different
// families.

func TestAutoscalingTagsToMap_TagDescriptionShape(t *testing.T) {
	// autoscaling / SSM use ec2types.TagDescription, not Tag.
	got := autoscalingTagsToMap([]astypes.TagDescription{
		{Key: aws.String("team"), Value: aws.String("infra")},
		{Key: aws.String("empty"), Value: nil},
		{Key: nil, Value: aws.String("dropped")},
	})
	assert.Equal(t, map[string]any{"team": "infra", "empty": ""}, got)
}

func TestEmrTagsToMap_StringMapShape(t *testing.T) {
	got := emrTagsToMap([]emrtypes.Tag{
		{Key: aws.String("k"), Value: aws.String("v")},
	})
	// must be a map[string]string, and key/value must not be swapped
	assert.Equal(t, map[string]string{"k": "v"}, got)
}

func TestTagsToStringMap(t *testing.T) {
	t.Run("nil slice returns non-nil empty map", func(t *testing.T) {
		got := tagsToStringMap(nil, kvKey, kvValue)
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("returns a map[string]string with the same nil policy", func(t *testing.T) {
		got := tagsToStringMap([]keyValueTag{
			{Key: aws.String("a"), Value: aws.String("1")},
			{Key: nil, Value: aws.String("drop")},
			{Key: aws.String("b"), Value: nil},
		}, kvKey, kvValue)
		assert.Equal(t, map[string]string{"a": "1", "b": ""}, got)
	})
}
