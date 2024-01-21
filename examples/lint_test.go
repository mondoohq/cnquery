// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package examples

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/internal/bundle"
	"testing"
)

func TestExampleLint(t *testing.T) {
	queryPackBundle, err := explorer.BundleFromPaths(".")
	require.NoError(t, err)

	lintErr := bundle.Lint(queryPackBundle)
	assert.Equal(t, []string{}, lintErr)
}
