// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientStatus_ConfigFileSerialization(t *testing.T) {
	tests := []struct {
		name         string
		configFile   string
		expectKey    bool
		expectedPath string
	}{
		{
			name:         "with a config file path",
			configFile:   "/home/user/.config/mondoo/mondoo.yml",
			expectKey:    true,
			expectedPath: "/home/user/.config/mondoo/mondoo.yml",
		},
		{
			name:       "without a config file path",
			configFile: "",
			expectKey:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(ClientStatus{ConfigFile: tt.configFile})
			require.NoError(t, err)

			var decoded map[string]any
			require.NoError(t, json.Unmarshal(data, &decoded))

			value, present := decoded["configFile"]
			if tt.expectKey {
				assert.True(t, present, "configFile should be present in JSON output when set")
				assert.Equal(t, tt.expectedPath, value, "configFile should round-trip the path")
			} else {
				assert.False(t, present, "configFile should be omitted from JSON output when empty")
			}
		})
	}
}
