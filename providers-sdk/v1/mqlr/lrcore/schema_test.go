// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lrcore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/types"
)

const provider = "🐱"

func schemaFor(t *testing.T, s string) *resources.Schema {
	ast := parse(t, s)
	ast.Options = map[string]string{"provider": provider}
	res, err := Schema(ast)
	require.NoError(t, err)
	return res
}

func TestSchema(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		res := schemaFor(t, "")
		assert.Empty(t, res.Resources)
	})

	t.Run("chain resource creation", func(t *testing.T) {
		res := schemaFor(t, `
			platform.has.name {
				str string
				comp() string
				}
		`)
		require.NotEmpty(t, res.Resources)
		expectedPlatform := &resources.ResourceInfo{
			Id:          "platform",
			IsExtension: true,
			Fields: map[string]*resources.Field{
				"has": {
					Name:               "has",
					Type:               string(types.Resource("platform.has")),
					Provider:           provider,
					IsImplicitResource: true,
				},
			},
		}
		expectedPlatforHas := &resources.ResourceInfo{
			Id:          "platform.has",
			IsExtension: true,
			Fields: map[string]*resources.Field{
				"name": {
					Name:               "name",
					Type:               string(types.Resource("platform.has.name")),
					Provider:           provider,
					IsImplicitResource: true,
				},
			},
		}
		expectedPlatformHasName := &resources.ResourceInfo{
			Id:       "platform.has.name",
			Provider: provider,
			Name:     "platform.has.name",
			Fields: map[string]*resources.Field{
				"str": {
					Name: "str",
					Type: string(types.String),
					// is mandatory because its's static (not computed)
					IsMandatory: true,
					Refs:        []string{},
					Provider:    provider,
				},
				"comp": {
					Name:     "comp",
					Type:     string(types.String),
					Refs:     []string{},
					Provider: provider,
				},
			},
		}
		assert.Equal(t, expectedPlatform, res.Resources["platform"])
		assert.Equal(t, expectedPlatforHas, res.Resources["platform.has"])
		assert.Equal(t, expectedPlatformHasName, res.Resources["platform.has.name"])
	})
}

func TestDetermnisticSchema(t *testing.T) {
	lrSchema, err := os.ReadFile("testdata/new.lr")
	require.NoError(t, err)
	ast := parse(t, string(lrSchema))
	schema, err := Schema(ast)
	require.NoError(t, err)
	for range 100 {
		newAst := parse(t, string(lrSchema))
		newSchema, err := Schema(newAst)
		require.NoError(t, err)
		require.Equal(t, schema, newSchema)
	}
}
