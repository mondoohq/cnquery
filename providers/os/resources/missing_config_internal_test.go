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
	"go.mondoo.com/mql/v13/utils/syncx"
)

func newMissingConfigTestRuntime(t *testing.T) *plugin.Runtime {
	t.Helper()

	conn, err := mock.New(0, &inventory.Asset{
		Platform: &inventory.Platform{
			Name:    "debian",
			Version: "10",
			Family:  []string{"debian", "linux"},
		},
	})
	require.NoError(t, err)

	return &plugin.Runtime{
		Connection: conn,
		Resources:  &syncx.Map[plugin.Resource]{},
	}
}

func TestNtpConfMissingFileReturnsEmptyData(t *testing.T) {
	runtime := newMissingConfigTestRuntime(t)
	conf := &mqlNtpConf{MqlRuntime: runtime}

	content := conf.GetContent()
	require.NoError(t, content.Error)
	assert.Empty(t, content.Data)
	assert.Equal(t, plugin.StateIsSet, content.State)

	settings := conf.GetSettings()
	require.NoError(t, settings.Error)
	assert.Empty(t, settings.Data)
	assert.Equal(t, plugin.StateIsSet, settings.State)

	servers := conf.GetServers()
	require.NoError(t, servers.Error)
	assert.Empty(t, servers.Data)
	assert.Equal(t, plugin.StateIsSet, servers.State)

	restrict := conf.GetRestrict()
	require.NoError(t, restrict.Error)
	assert.Empty(t, restrict.Data)
	assert.Equal(t, plugin.StateIsSet, restrict.State)

	fudge := conf.GetFudge()
	require.NoError(t, fudge.Error)
	assert.Empty(t, fudge.Data)
	assert.Equal(t, plugin.StateIsSet, fudge.State)
}

func TestParseIniMissingFileReturnsEmptyData(t *testing.T) {
	runtime := newMissingConfigTestRuntime(t)
	raw, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData("/etc/security/pwquality.conf"),
	})
	require.NoError(t, err)

	ini := &mqlParseIni{
		MqlRuntime: runtime,
		File: plugin.TValue[*mqlFile]{
			Data:  raw.(*mqlFile),
			State: plugin.StateIsSet,
		},
		Delimiter: plugin.TValue[string]{
			Data:  "=",
			State: plugin.StateIsSet,
		},
	}

	content := ini.GetContent()
	require.NoError(t, content.Error)
	assert.Empty(t, content.Data)
	assert.Equal(t, plugin.StateIsSet, content.State)

	sections := ini.GetSections()
	require.NoError(t, sections.Error)
	assert.Empty(t, sections.Data)
	assert.Equal(t, plugin.StateIsSet, sections.State)

	params := ini.GetParams()
	require.NoError(t, params.Error)
	assert.Empty(t, params.Data)
	assert.Equal(t, plugin.StateIsSet, params.State)
}
