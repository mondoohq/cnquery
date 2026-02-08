// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package llx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/exec"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func TestResourceMapV2(t *testing.T) {
	t.Run("ArrayLike not empty", func(t *testing.T) {
		result, err := exec.Exec(`users.where(group != empty).map(name).all(_ != empty)`, testutils.LinuxMock(), testutils.Features, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, true, result.Value)
	})
	t.Run("ArrayLike empty", func(t *testing.T) {
		result, err := exec.Exec(`users.where(group == empty).map(name).all(_ != empty)`, testutils.LinuxMock(), testutils.Features, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, true, result.Value)
	})
}
