// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestMacOSMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/osx_test.toml")
	provider, err := mock.New(0, filepath, &inventory.Asset{})
	require.NoError(t, err)

	lid := MacOSIdProvider{connection: provider}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "5c09e2c7-07f2-5bee-be82-7cb70688e55c", id, "machine id is properly detected")
}
