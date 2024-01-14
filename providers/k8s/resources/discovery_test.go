// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

func TestConvertImagesToAssets(t *testing.T) {
	images := map[string]ContainerImage{
		"nginx:1.25.3": {
			resolvedImage: "nginx@sha256:10d1f5b58f74683ad34eb29287e07dab1e90f10af243f151bb50aa5dbb4d62ee",
		},
	}
	expectedAssets := []inventory.Asset{
		{
			Name: "index.docker.io/library/nginx@10d1f5b58f74",
		},
	}

	assets, err := convertImagesToAssets(images)
	require.NoError(t, err)
	require.Len(t, assets, len(images))

	for i := range assets {
		require.NotNil(t, assets[i])
		require.Equal(t, expectedAssets[i].Name, assets[i].Name)
	}
}

func TestSetNamespaceFilters(t *testing.T) {
	cfg := &inventory.Config{
		Options: map[string]string{
			"namespaces":         "namespace1,namespace2",
			"namespaces-exclude": "namespace3,namespace4",
		},
	}

	nsFilter := setNamespaceFilters(cfg)

	expectedInclude := []string{"namespace1", "namespace2"}
	expectedExclude := []string{"namespace3", "namespace4"}

	require.Equal(t, expectedInclude, nsFilter.include)
	require.Equal(t, expectedExclude, nsFilter.exclude)

	// missing "s" in namespaces
	cfg = &inventory.Config{
		Options: map[string]string{
			"namespace":         "namespace1,namespace2",
			"namespace-exclude": "namespace3,namespace4",
		},
	}

	nsFilter = setNamespaceFilters(cfg)

	require.Nil(t, nsFilter.include)
	require.Nil(t, nsFilter.exclude)
}
