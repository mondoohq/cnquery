// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/api/groupssettings/v1"
)

func TestParseGroupSettingBool(t *testing.T) {
	require.True(t, parseGroupSettingBool("true"))
	require.True(t, parseGroupSettingBool("True")) // API casing varies
	require.True(t, parseGroupSettingBool("TRUE"))
	require.False(t, parseGroupSettingBool("false"))
	require.False(t, parseGroupSettingBool(""))
	require.False(t, parseGroupSettingBool("yes")) // only "true" is truthy
}

func TestGroupSettingsToData(t *testing.T) {
	settings := &groupssettings.Groups{
		WhoCanJoin:                 "INVITED_CAN_JOIN",
		WhoCanViewMembership:       "ALL_MEMBERS_CAN_VIEW",
		WhoCanPostMessage:          "ALL_MEMBERS_CAN_POST",
		MessageModerationLevel:     "MODERATE_NONE",
		SpamModerationLevel:        "MODERATE",
		AllowExternalMembers:       "true",
		AllowWebPosting:            "false",
		ArchiveOnly:                "false",
		IsArchived:                 "true",
		IncludeInGlobalAddressList: "true",
		EnableCollaborativeInbox:   "false",
	}

	d := groupSettingsToData(settings)

	// enum strings pass through unchanged
	require.Equal(t, "INVITED_CAN_JOIN", d.WhoCanJoin)
	require.Equal(t, "ALL_MEMBERS_CAN_VIEW", d.WhoCanViewMembership)
	require.Equal(t, "ALL_MEMBERS_CAN_POST", d.WhoCanPostMessage)
	require.Equal(t, "MODERATE_NONE", d.MessageModerationLevel)
	require.Equal(t, "MODERATE", d.SpamModerationLevel)

	// "true"/"false" strings normalize to typed bools
	require.True(t, d.AllowExternalMembers)
	require.False(t, d.AllowWebPosting)
	require.False(t, d.ArchiveOnly)
	require.True(t, d.IsArchived)
	require.True(t, d.IncludeInGlobalAddressList)
	require.False(t, d.EnableCollaborativeInbox)
	// unset boolean field defaults to false
	require.False(t, d.MembersCanPostAsTheGroup)
}

func TestHasRecovery(t *testing.T) {
	require.True(t, hasRecovery("user@example.com", ""))
	require.True(t, hasRecovery("", "+15555550123"))
	require.True(t, hasRecovery("user@example.com", "+15555550123"))
	require.False(t, hasRecovery("", ""))
}
