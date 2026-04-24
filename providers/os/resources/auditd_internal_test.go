// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/utils/syncx"
)

func newAuditdRulesTestRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()

	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "oraclelinux",
			Version: "8",
			Family:  []string{"oraclelinux", "linux"},
		},
	})
	require.NoError(t, err)

	return &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
}

func TestAuditdRulesMissingPathReturnsEmptyLists(t *testing.T) {
	runtime := newAuditdRulesTestRuntime(t)
	rules := &mqlAuditdRules{MqlRuntime: runtime}

	controls := rules.GetControls()
	require.NoError(t, controls.Error)
	assert.Empty(t, controls.Data)
	assert.Equal(t, plugin.StateIsSet, controls.State)

	files := rules.GetFiles()
	require.NoError(t, files.Error)
	assert.Empty(t, files.Data)
	assert.Equal(t, plugin.StateIsSet, files.State)

	syscalls := rules.GetSyscalls()
	require.NoError(t, syscalls.Error)
	assert.Empty(t, syscalls.Data)
	assert.Equal(t, plugin.StateIsSet, syscalls.State)
}
