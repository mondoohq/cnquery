// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventoryloader

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestInventoryTemplate(t *testing.T) {
	os.Setenv("MY_APPLICATION", "app1")
	os.Setenv("OPERATING_ENVIRONMENT", "dev")
	inventoryTemplate := `
spec:
  assets:
    - name: Scenario TF {{ getenv "OPERATING_ENVIRONMENT" }}
      connections:
        - type: terraform-hcl
          options:
            ignore-dot-terraform: "false"
            path: okta.tf
      annotations:
        Application: {{ getenv "MY_APPLICATION" }}
        OperatingEnv: {{ getenv "OPERATING_ENVIRONMENT" }}
`

	data, err := renderTemplate([]byte(inventoryTemplate))
	require.NoError(t, err)

	assert.Contains(t, string(data), "Application: app1")
	assert.Contains(t, string(data), "Scenario TF dev")
}
