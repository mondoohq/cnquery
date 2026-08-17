// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// A tenant-level role is owned by no organization. The reference has to come
// back explicitly null rather than merely unresolved, because an unset field
// leaves the runtime believing the value was never computed.
func TestRoleOwnerIsNullWithoutOwningOrganization(t *testing.T) {
	empty := ""

	for _, test := range []struct {
		title   string
		ownerID *string
	}{
		{"absent owner_id", nil},
		{"empty owner_id", &empty},
	} {
		t.Run(test.title, func(t *testing.T) {
			role := &mqlAuth0Role{}
			role.cacheOwnerID = test.ownerID

			owner, err := role.owner()

			require.NoError(t, err)
			assert.Nil(t, owner)
			assert.Equal(t, plugin.StateIsSet|plugin.StateIsNull, role.Owner.State,
				"owner must be marked set and null, not left unset")
		})
	}
}
