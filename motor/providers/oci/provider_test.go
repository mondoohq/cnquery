//go:build debugtest
// +build debugtest

package oci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOciProvider(t *testing.T) {
	provider, err := New(nil)
	require.NoError(t, err)

	ctx := context.Background()
	regions, err := provider.GetRegions(ctx)
	require.NoError(t, err)
	assert.NotNil(t, regions)

	compartmentIds, err := provider.GetCompartments(ctx)
	require.NoError(t, err)
	assert.NotNil(t, compartmentIds)
}
