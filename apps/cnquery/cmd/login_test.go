// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCmd_ProvidersURLFlag(t *testing.T) {
	// Test that the providers-url flag is properly defined on the LoginCmd
	flag := LoginCmd.Flags().Lookup("providers-url")
	require.NotNil(t, flag, "providers-url flag should be defined")
	assert.Equal(t, "", flag.DefValue, "providers-url default value should be empty")
	assert.Equal(t, "string", flag.Value.Type(), "providers-url should be a string flag")
}

func TestProvidersURLViperSet(t *testing.T) {
	tests := []struct {
		name              string
		providersURL      string
		expectInConfig    bool
		expectedConfigVal string
	}{
		{
			name:              "with valid providers-url",
			providersURL:      "https://my-custom-provider.com",
			expectInConfig:    true,
			expectedConfigVal: "https://my-custom-provider.com",
		},
		{
			name:           "with empty providers-url",
			providersURL:   "",
			expectInConfig: false,
		},
		{
			name:              "with custom path providers-url",
			providersURL:      "https://internal.example.com/providers",
			expectInConfig:    true,
			expectedConfigVal: "https://internal.example.com/providers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper before each test
			viper.Reset()

			// Simulate the behavior in the register function
			if tt.providersURL != "" {
				viper.Set("providers_url", tt.providersURL)
			}

			// Verify the viper state
			if tt.expectInConfig {
				assert.True(t, viper.IsSet("providers_url"), "providers_url should be set in viper")
				assert.Equal(t, tt.expectedConfigVal, viper.GetString("providers_url"))
			} else {
				assert.False(t, viper.IsSet("providers_url"), "providers_url should not be set in viper")
			}
		})
	}
}

func TestLoginCmd_GetProvidersURLFromFlag(t *testing.T) {
	// Reset any previous flag values
	err := LoginCmd.Flags().Set("providers-url", "")
	require.NoError(t, err)

	// Test setting the flag value
	err = LoginCmd.Flags().Set("providers-url", "https://custom-provider.example.com")
	require.NoError(t, err)

	// Retrieve the value
	providersURL, err := LoginCmd.Flags().GetString("providers-url")
	require.NoError(t, err)
	assert.Equal(t, "https://custom-provider.example.com", providersURL)
	// Reset the flag for other tests
	err = LoginCmd.Flags().Set("providers-url", "")
	require.NoError(t, err)
}

func TestLoginCmd_AllFlags(t *testing.T) {
	// Verify all expected flags are present on LoginCmd
	expectedFlags := []struct {
		name         string
		shorthand    string
		defaultValue string
		flagType     string
	}{
		{"token", "t", "", "string"},
		{"annotation", "", "[]", "stringToString"},
		{"providers-url", "", "", "string"},
		{"name", "", "", "string"},
		{"api-endpoint", "", "", "string"},
		{"timer", "", "0", "int"},
		{"splay", "", "0", "int"},
	}

	for _, ef := range expectedFlags {
		t.Run(ef.name, func(t *testing.T) {
			flag := LoginCmd.Flags().Lookup(ef.name)
			require.NotNil(t, flag, "flag %s should be defined", ef.name)
			assert.Equal(t, ef.defaultValue, flag.DefValue, "flag %s default value mismatch", ef.name)
			assert.Equal(t, ef.flagType, flag.Value.Type(), "flag %s type mismatch", ef.name)
			if ef.shorthand != "" {
				assert.Equal(t, ef.shorthand, flag.Shorthand, "flag %s shorthand mismatch", ef.name)
			}
		})
	}
}
