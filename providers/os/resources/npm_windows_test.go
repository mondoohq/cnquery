// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Windows paths only glob on Windows: elsewhere a backslash is an escape
// character, so this test lives in a _windows_test.go file.
func TestCollectNpmPackagesInPaths_windowsGlobalPrefixes(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	writeNpmManifest(t, mockFS, `C:\Users\alice\AppData\Roaming\npm\node_modules\@openai\codex`, "@openai/codex", "0.38.0")
	writeNpmManifest(t, mockFS, `C:\Windows\System32\config\systemprofile\AppData\Roaming\npm\node_modules\@github\copilot`, "@github/copilot", "1.0.42")

	r := npmTestRuntime(t, mockFS)
	direct, transitive, _, err := collectNpmPackagesInPaths(r, mockFS, defaultNpmPaths)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"@openai/codex@0.38.0", "@github/copilot@1.0.42"}, append(packageNames(direct), packageNames(transitive)...))
}
