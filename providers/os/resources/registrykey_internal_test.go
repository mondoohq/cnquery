// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// When a registrykey.property is created without its fields pre-populated by
// initRegistrykeyProperty — e.g. replaying a recording that did not capture
// them — the compute fallbacks must fail cleanly (false / empty / null) rather
// than erroring, so that policies querying a missing property fail gracefully
// instead of erroring the whole check.
func TestRegistrykeyProperty_FallbacksFailGracefully(t *testing.T) {
	p := &mqlRegistrykeyProperty{}

	exists, err := p.exists()
	require.NoError(t, err)
	require.False(t, exists)

	data, err := p.data()
	require.NoError(t, err)
	require.Nil(t, data)

	val, err := p.value()
	require.NoError(t, err)
	require.Equal(t, "", val)

	typ, err := p.compute_type()
	require.NoError(t, err)
	require.Equal(t, "", typ)
}
