// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacUpdatesParser(t *testing.T) {
	f, err := os.Open("./testdata/com.apple.SoftwareUpdate.plist")
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}

	m, err := ParseSoftwarePlistUpdates(f)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(m), "detected the right amount of updates")

	pkg, err := findKb(m, "MSU_UPDATE_21G217_patch_12.6.1")
	require.NoError(t, err)
	assert.Equal(t, "MSU_UPDATE_21G217_patch_12.6.1", pkg.Name, "update detected")
	assert.Equal(t, "macOS Monterey 12.6.1", pkg.Description, "update title detected")
}
