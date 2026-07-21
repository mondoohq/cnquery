// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/stretchr/testify/assert"
)

// TestSimpleConfigurationSettingValue_SecretRedaction locks in the type-switch
// ordering: the concrete secret type also satisfies the string-setting-value
// interface, so if the String case is matched first a secret value is returned
// in cleartext instead of the "***" mask. The Secret case must win.
func TestSimpleConfigurationSettingValue_SecretRedaction(t *testing.T) {
	secret := betamodels.NewDeviceManagementConfigurationSecretSettingValue()
	secret.SetValue(ptr("super-secret-value"))
	assert.Equal(t, "***", simpleConfigurationSettingValue(secret),
		"secret setting values must be masked, never returned in cleartext")

	str := betamodels.NewDeviceManagementConfigurationStringSettingValue()
	str.SetValue(ptr("plain-value"))
	assert.Equal(t, "plain-value", simpleConfigurationSettingValue(str))

	integer := betamodels.NewDeviceManagementConfigurationIntegerSettingValue()
	integer.SetValue(ptr(int32(42)))
	assert.Equal(t, int64(42), simpleConfigurationSettingValue(integer))

	assert.Nil(t, simpleConfigurationSettingValue(nil))
}
