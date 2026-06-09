// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func TestHttpHeaderServer(t *testing.T) {
	t.Run("returns the Server header value", func(t *testing.T) {
		h := &mqlHttpHeader{
			Params: plugin.TValue[map[string]any]{
				Data:  map[string]any{"Server": []any{"nginx"}},
				State: plugin.StateIsSet,
			},
		}
		v, err := h.server()
		require.NoError(t, err)
		assert.Equal(t, "nginx", v)
	})

	t.Run("is null when the Server header is absent", func(t *testing.T) {
		h := &mqlHttpHeader{
			Params: plugin.TValue[map[string]any]{
				Data:  map[string]any{},
				State: plugin.StateIsSet,
			},
		}
		v, err := h.server()
		require.NoError(t, err)
		assert.Equal(t, "", v)
		assert.NotEqual(t, 0, h.Server.State&plugin.StateIsNull)
	})
}
