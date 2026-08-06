// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	workspacestypes "github.com/aws/aws-sdk-go-v2/service/workspaces/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== client properties are absent =====

// A directory the caller cannot read client properties for reports the defaults
// rather than an error, so a fleet-wide query is not lost to one directory.
func TestDirectoryClientPropertiesAbsent(t *testing.T) {
	d := &mqlAwsWorkspacesDirectory{}
	d.clientPropsFetched.Store(true) // fetched, but nothing came back

	policy, err := d.clientExperiencePolicy()
	require.NoError(t, err)
	assert.Equal(t, "", policy)

	reconnect, err := d.clientReconnectEnabled()
	require.NoError(t, err)
	assert.False(t, reconnect)

	logUpload, err := d.clientLogUploadEnabled()
	require.NoError(t, err)
	assert.False(t, logUpload)
}

// ===== client properties are present =====

func TestDirectoryClientPropertiesPresent(t *testing.T) {
	policy := "FORCE_UI_2026"
	d := &mqlAwsWorkspacesDirectory{}
	d.clientProps = &workspacestypes.ClientProperties{
		// the SDK models the policy as a *string, unlike the two enums beside it
		ClientExperiencePolicy: &policy,
		ReconnectEnabled:       workspacestypes.ReconnectEnumEnabled,
		LogUploadEnabled:       workspacestypes.LogUploadEnumEnabled,
	}
	d.clientPropsFetched.Store(true)

	got, err := d.clientExperiencePolicy()
	require.NoError(t, err)
	assert.Equal(t, "FORCE_UI_2026", got)

	reconnect, err := d.clientReconnectEnabled()
	require.NoError(t, err)
	assert.True(t, reconnect)

	logUpload, err := d.clientLogUploadEnabled()
	require.NoError(t, err)
	assert.True(t, logUpload)
}

// The enums carry a disabled value of their own, which must not be read as the
// zero value the absent case already reports.
func TestDirectoryClientPropertiesDisabled(t *testing.T) {
	d := &mqlAwsWorkspacesDirectory{}
	d.clientProps = &workspacestypes.ClientProperties{
		ReconnectEnabled: workspacestypes.ReconnectEnumDisabled,
		LogUploadEnabled: workspacestypes.LogUploadEnumDisabled,
	}
	d.clientPropsFetched.Store(true)

	reconnect, err := d.clientReconnectEnabled()
	require.NoError(t, err)
	assert.False(t, reconnect)

	logUpload, err := d.clientLogUploadEnabled()
	require.NoError(t, err)
	assert.False(t, logUpload)

	// a nil policy pointer must not panic
	policy, err := d.clientExperiencePolicy()
	require.NoError(t, err)
	assert.Equal(t, "", policy)
}
