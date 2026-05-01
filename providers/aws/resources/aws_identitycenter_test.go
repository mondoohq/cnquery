// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// instance() returns null when the application's parent-instance pointer wasn't
// cached (e.g. the application was constructed via NewResource rather than
// listed under an instance).
func TestIdentitycenterApplicationInstanceNullWhenCacheEmpty(t *testing.T) {
	a := &mqlAwsIdentitycenterApplication{}
	got, err := a.instance()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, a.Instance.IsNull())
	assert.True(t, a.Instance.IsSet())
}
