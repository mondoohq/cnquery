// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package examples

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/internal/bundle"
	"go.mondoo.com/cnquery/v11/providers"
)

func ensureProviders(ids []string) error {
	for _, id := range ids {
		_, err := providers.EnsureProvider(providers.ProviderLookup{ID: id}, true, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	dir := ".lint-providers"
	providers.CustomProviderPath = dir
	providers.DefaultPath = dir

	err := ensureProviders([]string{
		"go.mondoo.com/cnquery/v9/providers/os",
		"go.mondoo.com/cnquery/v9/providers/k8s",
	})
	if err != nil {
		panic(err)
	}

	exitVal := m.Run()

	// cleanup custom provider path to ensure no leftovers and other tests are not affected
	err = os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func TestExampleLint(t *testing.T) {
	queryPackBundle, err := explorer.BundleFromPaths(".")
	require.NoError(t, err)

	lintErr := bundle.Lint(queryPackBundle)
	assert.Equal(t, []string{}, lintErr)
}
