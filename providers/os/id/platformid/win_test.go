// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package platformid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestGuidWindows(t *testing.T) {
	provider, err := mock.New(0, "./testdata/guid_windows.toml", nil)
	require.NoError(t, err)

	lid := WinIdProvider{connection: provider}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "6BAB78BE-4623-4705-924C-2B22433A4489", id)
}
