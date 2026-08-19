// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumSliceToAny(t *testing.T) {
	t.Run("empty slice returns empty", func(t *testing.T) {
		result := enumSliceToAny([]bedrocktypes.ModelModality{})
		assert.Empty(t, result)
	})

	t.Run("converts modalities to string slice", func(t *testing.T) {
		modalities := []bedrocktypes.ModelModality{"TEXT", "IMAGE", "EMBEDDING"}
		result := enumSliceToAny(modalities)
		assert.Equal(t, []any{"TEXT", "IMAGE", "EMBEDDING"}, result)
	})

	t.Run("converts customization types", func(t *testing.T) {
		types := []bedrocktypes.ModelCustomization{"FINE_TUNING", "CONTINUED_PRE_TRAINING"}
		result := enumSliceToAny(types)
		assert.Equal(t, []any{"FINE_TUNING", "CONTINUED_PRE_TRAINING"}, result)
	})

	t.Run("nil slice returns empty", func(t *testing.T) {
		var s []bedrocktypes.InferenceType
		result := enumSliceToAny(s)
		assert.Empty(t, result)
	})
}

// TestBedrockCustomModelArgsAreSettableFields pins every argument the custom
// model init passes to a field the generated schema can actually set.
// SetAllData rejects an unknown key outright, so a stale argument left behind
// by a schema change does not degrade a field - it fails the whole resource on
// every typed reference, because this init runs on every NewResource for the
// type.
func TestBedrockCustomModelArgsAreSettableFields(t *testing.T) {
	args := bedrockCustomModelArgs(
		aws.String("arn:aws:bedrock:us-east-1:123456789012:custom-model/anthropic.claude/abc"),
		aws.String("tuned-claude"),
		"us-east-1",
		bedrocktypes.CustomizationTypeFineTuning,
	)

	require.NotEmpty(t, args)
	for key := range args {
		if key == "__id" {
			continue
		}
		_, ok := setDataFields["aws.bedrock.customModel."+key]
		assert.True(t, ok, "aws.bedrock.customModel has no settable field %q", key)
	}

	// The base model reaches the resource through the typed baseModel
	// accessor, never as a raw ARN argument.
	assert.NotContains(t, args, "baseModelArn")
}
