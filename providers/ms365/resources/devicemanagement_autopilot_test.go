// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"github.com/stretchr/testify/assert"
)

func TestAutopilotOobeToDict_Nil(t *testing.T) {
	assert.Nil(t, autopilotOobeToDict(nil))
}

func TestAutopilotOobeToDict_AllFields(t *testing.T) {
	s := betamodels.NewOutOfBoxExperienceSettings()
	s.SetHidePrivacySettings(ptr(true))
	s.SetHideEULA(ptr(false))
	s.SetSkipKeyboardSelectionPage(ptr(true))
	s.SetHideEscapeLink(ptr(true))
	userType := betamodels.STANDARD_WINDOWSUSERTYPE
	s.SetUserType(&userType)
	deviceUsage := betamodels.SHARED_WINDOWSDEVICEUSAGETYPE
	s.SetDeviceUsageType(&deviceUsage)

	got := autopilotOobeToDict(s)
	assert.Equal(t, map[string]any{
		"hidePrivacySettings":       true,
		"hideEULA":                  false,
		"skipKeyboardSelectionPage": true,
		"hideEscapeLink":            true,
		"userType":                  "standard",
		"deviceUsageType":           "shared",
	}, got)
}

func TestAutopilotOobeToDict_OmitsNilFields(t *testing.T) {
	s := betamodels.NewOutOfBoxExperienceSettings()
	s.SetHideEULA(ptr(true))

	got := autopilotOobeToDict(s)
	assert.Equal(t, map[string]any{"hideEULA": true}, got)
	assert.NotContains(t, got, "hidePrivacySettings")
	assert.NotContains(t, got, "userType")
}
