// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// windows.defender.status and windows.defender.preferences are both field paths
// on windows.defender and registered resource names in their own right. Without
// an Init the dotted form instantiates the sub-resource, the parent's accessor
// never runs, and every field reads null - so a check reading "antivirus is
// enabled" off windows.defender.status.antivirusEnabled reports null on a host
// where Defender is running and protecting the machine.
func TestWindowsDefenderSingletonsAreReachableByTheirOwnPath(t *testing.T) {
	for _, path := range []string{"windows.defender.status", "windows.defender.preferences"} {
		t.Run(path, func(t *testing.T) {
			_, isField := getDataFields[path]
			require.True(t, isField, "%s should be a field path on windows.defender", path)

			factory, isResource := resourceFactories[path]
			require.True(t, isResource, "%s should also be a registered resource name", path)

			assert.NotNil(t, factory.Init,
				"%s resolves to the resource, not the field, so without an Init every field reads null", path)
		})
	}
}
