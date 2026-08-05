// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
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
	mu       sync.Mutex
	commands []string
}

func (c *optionalFeatureRecordingConnection) RunCommand(command string) (*shared.Command, error) {
	c.mu.Lock()
	c.commands = append(c.commands, command)
	c.mu.Unlock()
	return c.Connection.RunCommand(command)
}

func (c *optionalFeatureRecordingConnection) recorded() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.commands...)
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
	assert.Equal(t, []string{listCmd}, conn.recorded(), "the enumeration only lists features")

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

	assert.Equal(t, []string{listCmd, detailCmd}, conn.recorded(),
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

	_, res, err := initWindowsOptionalFeature(runtime, map[string]*llx.RawData{
		"name": llx.StringData("SMB1Protocol"),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{lookupCmd}, conn.recorded())

	feature, ok := res.(*mqlWindowsOptionalFeature)
	require.True(t, ok)
	assert.Equal(t, "SMB1Protocol", feature.GetName().Data)
	assert.Equal(t, "SMB 1.0/CIFS File Sharing Support", feature.GetDisplayName().Data)
	assert.Equal(t, int64(2), feature.GetState().Data)
	assert.True(t, feature.GetEnabled().Data)
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
	assert.Equal(t, []string{lookupCmd}, conn.recorded())

	for i := 0; i < 5; i++ {
		again, err := NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
			"name": llx.StringData("SMB1Protocol"),
		})
		require.NoError(t, err)
		assert.Same(t, first, again, "a repeated reference must reuse the cached feature")
	}
	assert.Equal(t, []string{lookupCmd}, conn.recorded(), "the image is queried once per scan, not once per reference")

	// a different feature is still its own lookup
	_, err = NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
		"name": llx.StringData("TelnetClient"),
	})
	require.Error(t, err, "unknown feature names still fail")
	assert.Len(t, conn.recorded(), 2)
}

// cnspec resolves filter and check entrypoints concurrently, so the cache lookup
// alone is not enough: every caller can miss it before the first one stores the
// resource. The lookup has to collapse into a single image query.
func TestInitOptionalFeatureCollapsesConcurrentLookups(t *testing.T) {
	lookupCmd := powershell.Encode(windows.OptionalFeatureQuery("SMB1Protocol"))

	mockConn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "windows",
			Family: []string{"windows"},
		},
	}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			lookupCmd: {Stdout: `{"FeatureName": "SMB1Protocol", "DisplayName": "SMB 1.0/CIFS File Sharing Support", "Description": "d", "State": 2}`},
		},
	}))
	require.NoError(t, err)

	conn := &optionalFeatureRecordingConnection{Connection: mockConn}
	runtime := &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}

	const callers = 16
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]plugin.Resource, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = NewResource(runtime, "windows.optionalFeature", map[string]*llx.RawData{
				"name": llx.StringData("SMB1Protocol"),
			})
		}(i)
	}
	close(start)
	wg.Wait()

	for i := range results {
		require.NoError(t, errs[i])
		assert.Same(t, results[0], results[i], "all callers must share one feature resource")
	}
	assert.Equal(t, []string{lookupCmd}, conn.recorded(),
		"concurrent references must collapse into a single image query")
}
