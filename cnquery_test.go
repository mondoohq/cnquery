// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cnquery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testGetLatestVersion(t *testing.T) {
	version, err := GetLatestVersion()

	assert.NoError(t, err)
	assert.NotNil(t, version)
}
