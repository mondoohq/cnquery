// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
	"go.mondoo.com/mql/v13/utils/syncx"
)

type optionalFeatureRecordingConnection struct {
	*mock.Connection
	commands []string
}

func (c *optionalFeatureRecordingConnection) RunCommand(command string) (*shared.Command, error) {
	c.commands = append(c.commands, command)
	return c.Connection.RunCommand(command)
}

// The enumeration must not pay for DISM's detailed per-feature lookup: every
// check that filters on a feature state runs it, and it costs tens of seconds on
// a Windows client. Display name and description are loaded on demand instead,
// once for the whole enumeration.
func TestOptionalFeaturesDefersDetailQuery(t *testing.T) {
	listCmd := powershell.Encode(windows.QUERY_OPTIONAL_FEATURES)
	detailCmd := powershell.Encode(windows.QUERY_OPTIONAL_FEATURE_DETAILS)

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			listCmd: {Stdout: `[
				{"FeatureName": "SMB1Protocol", "State": 2},
				{"FeatureName": "TelnetClient", "State": 0}
			]`},
			detailCmd: {Stdout: `[
				{"FeatureName": "SMB1Protocol", "DisplayName": "SMB 1.0/CIFS File Sharing Support", "Description": "Support for the SMB 1.0/CIFS file sharing protocol", "State": 2},
				{"FeatureName": "TelnetClient", "DisplayName": "Telnet Client", "Description": "Telnet Client uses the Telnet protocol", "State": 0}
			]`},
		},
	}))
	require.NoError(t, err)

	conn := &optionalFeatureRecordingConnection{Connection: mockConn}
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	features, err := (&mqlWindows{MqlRuntime: runtime}).optionalFeatures()
	require.NoError(t, err)
	require.Len(t, features, 2)
	assert.Equal(t, []string{listCmd}, conn.commands, "the enumeration only lists features")

	smb := features[0].(*mqlWindowsOptionalFeature)
	assert.Equal(t, "SMB1Protocol", smb.GetName().Data)
	assert.Equal(t, int64(2), smb.GetState().Data)
	assert.True(t, smb.GetEnabled().Data)

	telnet := features[1].(*mqlWindowsOptionalFeature)
	assert.Equal(t, "TelnetClient", telnet.GetName().Data)
	assert.False(t, telnet.GetEnabled().Data)

	displayName := smb.GetDisplayName()
	require.NoError(t, displayName.Error)
	assert.Equal(t, "SMB 1.0/CIFS File Sharing Support", displayName.Data)

	description := telnet.GetDescription()
	require.NoError(t, description.Error)
	assert.Equal(t, "Telnet Client uses the Telnet protocol", description.Data)

	assert.Equal(t, []string{listCmd, detailCmd}, conn.commands,
		"details are loaded once for the whole enumeration")
}

// A feature looked up by name never enumerates the image.
func TestInitOptionalFeatureUsesTargetedLookup(t *testing.T) {
	lookupCmd := powershell.Encode(windows.OptionalFeatureQuery("SMB1Protocol"))

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			lookupCmd: {Stdout: `{
				"FeatureName": "SMB1Protocol",
				"DisplayName": "SMB 1.0/CIFS File Sharing Support",
				"Description": "Support for the SMB 1.0/CIFS file sharing protocol",
				"State": 2
			}`},
		},
	}))
	require.NoError(t, err)

	conn := &optionalFeatureRecordingConnection{Connection: mockConn}
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	args, _, err := initWindowsOptionalFeature(runtime, map[string]*llx.RawData{
		"name": llx.StringData("SMB1Protocol"),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{lookupCmd}, conn.commands)
	assert.Equal(t, "SMB 1.0/CIFS File Sharing Support", args["displayName"].Value)
	assert.Equal(t, int64(2), args["state"].Value)
	assert.Equal(t, true, args["enabled"].Value)
}

// NewResource runs a resource's init before it consults the resource cache, so a
// resource whose init talks to the target must dedupe itself: a policy that
// filters on a feature state references it from several checks, and each of those
// is a separate NewResource call.
func TestInitOptionalFeatureDedupesAcrossReferences(t *testing.T) {
	lookupCmd := powershell.Encode(windows.OptionalFeatureQuery("SMB1Protocol"))

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			lookupCmd: {Stdout: `{
				"FeatureName": "SMB1Protocol",
				"DisplayName": "SMB 1.0/CIFS File Sharing Support",
				"Description": "Support for the SMB 1.0/CIFS file sharing protocol",
				"State": 2
			}`},
		},
	}))
	require.NoError(t, err)

	conn := &optionalFeatureRecordingConnection{Connection: mockConn}
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	first, err := NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
		"name": llx.StringData("SMB1Protocol"),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{lookupCmd}, conn.commands)

	for i := 0; i < 5; i++ {
		again, err := NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
			"name": llx.StringData("SMB1Protocol"),
		})
		require.NoError(t, err)
		assert.Same(t, first, again, "a repeated reference must reuse the cached feature")
	}
	assert.Equal(t, []string{lookupCmd}, conn.commands, "the image is queried once per scan, not once per reference")

	// a different feature is still its own lookup
	_, err = NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
		"name": llx.StringData("TelnetClient"),
	})
	require.Error(t, err, "unknown feature names still fail")
	assert.Len(t, conn.commands, 2)
}
