// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package generic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoEpoch(t *testing.T) {
	r := VersionWithoutEpoch("1632431095:1.2.2-r7")
	assert.Equal(t, "1.2.2-r7", r)
}
