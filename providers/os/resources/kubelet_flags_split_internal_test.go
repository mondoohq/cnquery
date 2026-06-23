// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeyValueListFlag(t *testing.T) {
	// Well-formed pairs parse; tokens without `=` (e.g. a trailing comma) are
	// skipped rather than panicking on an out-of-range index.
	res := parseKeyValueListFlag("a=1,malformed,b=2,")
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, res)
}

func TestMergeDeprecatedFlagsIntoConfig_FeatureGatesMalformed(t *testing.T) {
	cfg := map[string]any{}
	flags := map[string]any{"feature-gates": "Foo=true,malformed,Bar=false"}

	require.NotPanics(t, func() {
		err := mergeDeprecatedFlagsIntoConfig(cfg, flags)
		require.NoError(t, err)
	})

	fg, ok := cfg["featureGates"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "true", fg["Foo"])
	assert.Equal(t, "false", fg["Bar"])
	_, hasMalformed := fg["malformed"]
	assert.False(t, hasMalformed, "malformed token must be skipped")
}
