// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/types"
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
			platform.has.name {}
		`)
		require.NotEmpty(t, res.Resources)
		require.Equal(t, &resources.ResourceInfo{
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
		}, res.Resources["platform"])
	})
}
