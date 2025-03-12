// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config_test

import (
	"path/filepath"
	"testing"

	subject "go.mondoo.com/cnquery/v11/cli/config"
	"gopkg.in/yaml.v2"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)
	tempDir := "/tmp/config_test"
	configPath := filepath.Join(tempDir, "config.yaml")
	viper.SetConfigFile(configPath)

	t.Run("creates new config file when missing", func(t *testing.T) {
		err := subject.StoreConfig()
		require.NoError(t, err)

		exists, err := afero.Exists(fs, configPath)
		require.NoError(t, err)
		assert.True(t, exists, "config file should be created")
	})

	t.Run("writes to existing config file", func(t *testing.T) {
		// Pre-create the config file
		assert.NoError(t, afero.WriteFile(fs, configPath, []byte("initial data"), 0o644))

		viper.Set("key", "value")
		err := subject.StoreConfig()
		require.NoError(t, err)

		// Validate YAML format of the stored file
		content, err := afero.ReadFile(fs, configPath)
		assert.NoError(t, err, "Should be able to read the config file")

		var yamlData map[string]interface{}
		err = yaml.Unmarshal(content, &yamlData)
		assert.NoError(t, err, "Config file should be valid YAML")

		// Verify the saved key-value pair in YAML format
		assert.Equal(t, "value", yamlData["key"], "Config should retain stored values")
	})

	t.Run("correctly omits writing fields that we don't want to store", func(t *testing.T) {
		// valid
		viper.Set("key", "value")
		// omit field
		viper.Set("force", false)

		// store config
		err := subject.StoreConfig()
		require.NoError(t, err)

		// Validate no force field is written to disk
		content, err := afero.ReadFile(fs, configPath)
		assert.NoError(t, err)

		var yamlData map[string]interface{}
		err = yaml.Unmarshal(content, &yamlData)
		assert.NoError(t, err, "Config file should be valid YAML")

		// Verify valid field
		assert.Equal(t, "value", yamlData["key"], "should store valid field")
		// Verify invalid field
		_, exist := yamlData["force"]
		assert.False(t, exist, "should not store omitted field")
	})

	t.Run("handles error when failing to write config file", func(t *testing.T) {
		readOnlyFs := afero.NewReadOnlyFs(fs)
		viper.SetFs(readOnlyFs)

		err := subject.StoreConfig()
		assert.Error(t, err)
	})
}
