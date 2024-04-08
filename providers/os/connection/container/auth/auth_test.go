// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructKeychain(t *testing.T) {
	t.Run("default keychain only", func(t *testing.T) {
		keychain := getKeychains("test")
		require.Equal(t, 1, len(keychain))
	})
	t.Run("default keychain and ecr keychain", func(t *testing.T) {
		keychain := getKeychains("0000000000.dkr.ecr.us-east-1.amazonaws.com/test")
		require.Equal(t, 2, len(keychain))
	})

	t.Run("default keychain and acr keychain", func(t *testing.T) {
		keychain := getKeychains("test.azurecr.io")
		require.Equal(t, 2, len(keychain))
	})
}
