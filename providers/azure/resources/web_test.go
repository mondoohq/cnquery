// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	assert.False(t, isPlatformEol("python", ""))
	assert.False(t, isPlatformEol("python", "3.7"))
	assert.False(t, isPlatformEol("node", "12-lts"))
	assert.False(t, isPlatformEol("node", "10-lts"))
	assert.True(t, isPlatformEol("node", "11.1"))
	assert.True(t, isPlatformEol("node", "6.1"))
}
