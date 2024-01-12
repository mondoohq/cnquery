// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package platformid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestLinuxMachineId(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/linux_test.toml")
	provider, err := mock.New(filepath, nil)
	require.NoError(t, err)

	lid := LinuxIdProvider{connection: provider}
	id, err := lid.ID()
	require.NoError(t, err)

	assert.Equal(t, "39827700b8d246eb9446947c573ecff2", id, "machine id is properly detected")
}
