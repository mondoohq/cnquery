// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYumBoolParam(t *testing.T) {
	params := map[string]any{
		"gpgcheck":          "1",
		"localpkg_gpgcheck": "true",
		"repo_gpgcheck":     "0",
		"best":              "yes",
		"installonly_limit": "3",
		"plugins":           "On",
	}

	require.True(t, yumBoolParam(params, "gpgcheck"))
	require.True(t, yumBoolParam(params, "localpkg_gpgcheck"))
	require.True(t, yumBoolParam(params, "best"))
	require.True(t, yumBoolParam(params, "plugins"))

	require.False(t, yumBoolParam(params, "repo_gpgcheck"))
	require.False(t, yumBoolParam(params, "installonly_limit"))
	// absent directive is false
	require.False(t, yumBoolParam(params, "clean_requirements_on_remove"))
}
