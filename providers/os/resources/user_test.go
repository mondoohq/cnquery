// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/local"
	"go.mondoo.com/mql/utils/syncx"
)

// sshd and guest on Alpine have /dev/null as their home. Looking for keys
// below it fails with ENOTDIR, which is no keys, not an error.
func TestUserKeysWithAHomeThatIsAFile(t *testing.T) {
	home := filepath.Join(t.TempDir(), "null")
	require.NoError(t, os.WriteFile(home, nil, 0o644))

	conn := local.NewConnection(0, &inventory.Config{}, &inventory.Asset{Platform: &inventory.Platform{Name: "alpine", Family: []string{"linux", "unix"}}})
	rt := &plugin.Runtime{Connection: conn, Resources: &syncx.Map[plugin.Resource]{}}
	u := &mqlUser{MqlRuntime: rt}
	u.Home = plugin.TValue[string]{Data: home, State: plugin.StateIsSet}

	keys, err := u.sshkeys()
	require.NoError(t, err)
	assert.Empty(t, keys)

	ak, err := u.authorizedkeys(home)
	require.NoError(t, err)
	list := ak.GetList()
	require.NoError(t, list.Error)
	assert.Empty(t, list.Data)
}
