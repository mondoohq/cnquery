// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package exec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/exec"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

// A typed asset reference must keep behaving as an asset value at execution
// time: the root rides on the type, so the nil comparisons - the only thing MQL
// could ever do with an asset - have to keep dispatching on the asset type
// rather than falling through to a missing-function error. See ADR 031.
func TestAssetReferenceExec(t *testing.T) {
	rt := testutils.LinuxMock()

	res, err := exec.Exec("muser.running != null", rt, testutils.Features, nil)
	require.NoError(t, err)
	require.NoError(t, res.Error)
	assert.Equal(t, true, res.Value)

	res, err = exec.Exec("muser.running == null", rt, testutils.Features, nil)
	require.NoError(t, err)
	require.NoError(t, res.Error)
	assert.Equal(t, false, res.Value)
}

// Resolving into the referenced asset is not implemented yet: it runs host-side,
// where the coordinator and the recording layer are, and llx can reach neither.
// The chain compiles, so the failure has to name the reason at the deref rather
// than surface as a missing function on the asset type.
//
// Replace this with the resolution it describes when the backend lands (ADR 031
// phase 3).
func TestAssetChainNotResolvableYet(t *testing.T) {
	rt := testutils.LinuxMock()

	res, err := exec.Exec("muser.running.name", rt, testutils.Features, nil)
	require.NoError(t, err, "the chain compiles")
	require.Error(t, res.Error)
	assert.Contains(t, res.Error.Error(), "cross-asset resolution is not implemented yet")
	assert.Contains(t, res.Error.Error(), "muser", "the error names the asset it could not reach")
}
