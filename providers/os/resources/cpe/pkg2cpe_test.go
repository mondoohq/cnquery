// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPkg2Gen(t *testing.T) {
	cpe, err := NewPackage2Cpe("tar", "tar", "1.34+dfsg-1", "", "")
	require.NoError(t, err)
	assert.Equal(t, "cpe:2.3:a:tar:tar:1.34\\+dfsg-1:*:*:*:*:*:*:*", cpe)
}
