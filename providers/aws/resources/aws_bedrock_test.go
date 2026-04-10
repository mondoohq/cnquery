// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/stretchr/testify/assert"
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
