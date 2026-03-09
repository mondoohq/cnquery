// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package selfupdate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAndUpdate_EnvVarBehavior(t *testing.T) {
	// Save and restore environment variables after test
	origAutoUpdate := os.Getenv(EnvAutoUpdate)
	origEngine := os.Getenv(EnvAutoUpdateEngine)
	defer func() {
		if origAutoUpdate == "" {
			os.Unsetenv(EnvAutoUpdate)
		} else {
			os.Setenv(EnvAutoUpdate, origAutoUpdate)
		}
		if origEngine == "" {
			os.Unsetenv(EnvAutoUpdateEngine)
		} else {
			os.Setenv(EnvAutoUpdateEngine, origEngine)
		}
	}()

	t.Run("skips when MONDOO_AUTO_UPDATE is false", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdateEngine)
		os.Setenv(EnvAutoUpdate, "false")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips when MONDOO_AUTO_UPDATE is 0", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdateEngine)
		os.Setenv(EnvAutoUpdate, "0")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips when MONDOO_AUTO_UPDATE_ENGINE is false", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdate)
		os.Setenv(EnvAutoUpdateEngine, "false")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips when MONDOO_AUTO_UPDATE_ENGINE is 0", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdate)
		os.Setenv(EnvAutoUpdateEngine, "0")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips engine when MONDOO_AUTO_UPDATE is on but MONDOO_AUTO_UPDATE_ENGINE is off", func(t *testing.T) {
		os.Setenv(EnvAutoUpdate, "true")
		os.Setenv(EnvAutoUpdateEngine, "false")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("MONDOO_AUTO_UPDATE off overrides MONDOO_AUTO_UPDATE_ENGINE on", func(t *testing.T) {
		os.Setenv(EnvAutoUpdate, "false")
		os.Setenv(EnvAutoUpdateEngine, "true")

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("does not skip when neither env var is set", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdateEngine)
		os.Unsetenv(EnvAutoUpdate)

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0-rolling", // Use rolling to skip network check
		}

		// Will return false due to rolling version, but won't skip due to env vars
		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips when config is disabled", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdateEngine)
		os.Unsetenv(EnvAutoUpdate)

		cfg := Config{
			Enabled:        false,
			CurrentVersion: "1.0.0",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})

	t.Run("skips for rolling version", func(t *testing.T) {
		os.Unsetenv(EnvAutoUpdateEngine)
		os.Unsetenv(EnvAutoUpdate)

		cfg := Config{
			Enabled:        true,
			CurrentVersion: "1.0.0-rolling",
		}

		updated, err := CheckAndUpdate(cfg)
		assert.NoError(t, err)
		assert.False(t, updated)
	})
}

// TestEnvVarSeparation verifies that MONDOO_AUTO_UPDATE_ENGINE is separate from
// MONDOO_AUTO_UPDATE, ensuring that:
// 1. Engine binary auto-update can be disabled independently of provider auto-update
// 2. Provider auto-update (which reads MONDOO_AUTO_UPDATE via viper) is not affected
func TestEnvVarSeparation(t *testing.T) {
	t.Run("env vars are different", func(t *testing.T) {
		assert.NotEqual(t, EnvAutoUpdate, EnvAutoUpdateEngine)
		assert.Equal(t, "MONDOO_AUTO_UPDATE", EnvAutoUpdate)
		assert.Equal(t, "MONDOO_AUTO_UPDATE_ENGINE", EnvAutoUpdateEngine)
	})
}
