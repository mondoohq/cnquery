// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/utils/sortx"
)

func genTestSchema(t *testing.T) *AssetUrlSchema {
	root, err := newAssetUrlSchema("technology")
	require.NoError(t, err)

	err = root.Add(&AssetUrlBranch{
		PathSegments: []string{"technology=aws"},
		Key:          "account",
		Values: map[string]*AssetUrlBranch{
			"*": {
				Key: "service",
				Values: map[string]*AssetUrlBranch{
					"ec2": {
						References: []string{"technology=os"},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	err = root.Add(&AssetUrlBranch{
		PathSegments: []string{"technology=os"},
		Key:          "family",
		Values: map[string]*AssetUrlBranch{
			"windows": {
				Key: "platform",
				Values: map[string]*AssetUrlBranch{
					"windows server": nil,
				},
			},
		},
	})
	require.NoError(t, err)

	err = root.Add(&AssetUrlBranch{
		PathSegments: []string{"technology=os", "family=windows", "platform=windows server"},
		Key:          "version",
		Values: map[string]*AssetUrlBranch{
			"2019": nil,
			"2022": nil,
		},
	})
	require.NoError(t, err)

	return root
}

func TestAddSubtree(t *testing.T) {
	root := genTestSchema(t)

	t.Run("refresh cache and access keys", func(t *testing.T) {
		err := root.RefreshCache()
		require.NoError(t, err)

		keys := sortx.Keys(root.keys)
		assert.Equal(t, []string{
			"account", "family", "platform", "service", "technology", "version",
		}, keys)
	})

	t.Run("build a query with referenced tree positions", func(t *testing.T) {
		err := root.RefreshCache()
		require.NoError(t, err)

		queries := root.BuildQueries([]KV{
			{Key: "platform", Value: "windows server"},
		})
		assert.Equal(t, []AssetUrlChain{
			{
				KV{"technology", "aws"},
				KV{"account", "*"},
				KV{"service", "ec2"},
				KV{"family", "windows"},
				KV{"platform", "windows server"},
			},
			{
				KV{"technology", "os"},
				KV{"family", "windows"},
				KV{"platform", "windows server"},
			},
		}, queries)

		queries = root.BuildQueries([]KV{
			{Key: "account", Value: "123"},
			{Key: "platform", Value: "windows server"},
		})
		assert.Equal(t, []AssetUrlChain{
			{
				KV{"technology", "aws"},
				KV{"account", "*"},
				KV{"service", "ec2"},
				KV{"family", "windows"},
				KV{"platform", "windows server"},
			},
		}, queries)

		queries = root.BuildQueries([]KV{
			{Key: "technology", Value: "os"},
			{Key: "platform", Value: "windows server"},
		})
		assert.Equal(t, []AssetUrlChain{
			{
				KV{"technology", "os"},
				KV{"family", "windows"},
				KV{"platform", "windows server"},
			},
		}, queries)
	})
}

func TestBroken(t *testing.T) {
	root, err := NewAssetUrlSchema("technology")
	require.NoError(t, err)

	err = root.Add(&AssetUrlBranch{
		PathSegments: []string{"technology=tech1"},
		Key:          "environment",
		Values: map[string]*AssetUrlBranch{
			"*": {
				Key: "region",
				Values: map[string]*AssetUrlBranch{
					"*": nil,
				},
			},
		},
	})
	require.NoError(t, err)

	err = root.Add(&AssetUrlBranch{
		PathSegments: []string{"technology=tech2"},
		Key:          "environment",
		Values: map[string]*AssetUrlBranch{
			"*": nil,
		},
	})
	require.NoError(t, err)

	root.RefreshCache()

	queries := root.BuildQueries([]KV{
		{Key: "technology", Value: "tech2"},
		{Key: "environment", Value: "foo"},
	})

	require.Equal(t, []AssetUrlChain{
		{
			KV{"technology", "tech2"},
			KV{"environment", "foo"},
		},
	}, queries)

}
