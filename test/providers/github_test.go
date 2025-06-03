// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/test"
)

func TestGithubScanFlags(t *testing.T) {
	once.Do(setup)

	t.Run("github scan WITHOUT flags", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnquery", "scan", "github", "repo", "foo")
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 0, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"a valid GitHub authentication is required",
		)
	})
	t.Run("github scan WITH flags but missing app auth key", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnquery", "scan", "github", "repo", "foo",
			"--app-id", "123", "--app-installation-id", "456",
		)
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"could not parse private key", // expected! it means we loaded the flags
		)
	})
	t.Run("github scan WITH all required flags for app auth", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnquery", "scan", "github", "repo", "foo",
			"--app-id", "123", "--app-installation-id", "456", "--app-private-key", "private-key.pem",
		)
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"could not read private key", // expected! it means we loaded the flags
		)
	})
	t.Run("github scan with both auth methods, prefer app credentials", func(t *testing.T) {
		// NOTE this will fail but, it will load the flags and fail with the right message
		r := test.NewCliTestRunner("./cnquery", "scan", "github", "repo", "foo",
			// personal access token
			"--token", "abc",
			// application credentials
			"--app-id", "123", "--app-installation-id", "456", "--app-private-key", "private-key.pem",
		)
		err := r.Run()
		require.NoError(t, err)
		assert.Equal(t, 1, r.ExitCode())
		assert.NotNil(t, r.Stdout())
		assert.NotNil(t, r.Stderr())

		assert.Contains(t, string(r.Stderr()),
			"could not read private key", // expected! it means we use app credentials
		)
	})
}
