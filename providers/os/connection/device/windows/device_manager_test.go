// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOpts(t *testing.T) {
	t.Run("valid (only LUN)", func(t *testing.T) {
		opts := map[string]string{
			LunOption: "0",
		}
		err := validateOpts(opts)
		require.NoError(t, err)
	})

	t.Run("valid (only serial number)", func(t *testing.T) {
		opts := map[string]string{
			SerialNumberOption: "0",
		}
		err := validateOpts(opts)
		require.NoError(t, err)
	})

	t.Run("invalid (both LUN and serial number are provided", func(t *testing.T) {
		opts := map[string]string{
			SerialNumberOption: "1234",
			LunOption:          "1",
		}
		err := validateOpts(opts)
		require.Error(t, err)
	})

	t.Run("invalid (neither LUN nor serial number are provided", func(t *testing.T) {
		opts := map[string]string{}
		err := validateOpts(opts)
		require.Error(t, err)
	})
}
