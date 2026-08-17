// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Every rejection below has to happen before the connection is touched, so a
// bad argument reports what was wrong instead of panicking on a nil client.
func TestInitAuth0OrganizationRejectsBadArguments(t *testing.T) {
	for _, test := range []struct {
		title string
		args  map[string]*llx.RawData
	}{
		{"no id at all", map[string]*llx.RawData{}},
		{"id of the wrong type", map[string]*llx.RawData{"id": llx.IntData(42)}},
		{"empty id", map[string]*llx.RawData{"id": llx.StringData("")}},
	} {
		t.Run(test.title, func(t *testing.T) {
			args, res, err := initAuth0Organization(&plugin.Runtime{}, test.args)

			require.Error(t, err)
			assert.Nil(t, args)
			assert.Nil(t, res)
			assert.Contains(t, err.Error(), "auth0.organization")
		})
	}
}

// A caller that already holds a populated organization must not trigger a
// second read of it.
func TestInitAuth0OrganizationPassesThroughPopulatedResource(t *testing.T) {
	args := map[string]*llx.RawData{
		"id":          llx.StringData("org_abc"),
		"name":        llx.StringData("acme"),
		"displayName": llx.StringData("Acme"),
	}

	got, res, err := initAuth0Organization(&plugin.Runtime{}, args)

	require.NoError(t, err)
	assert.Nil(t, res, "no resource is built on the fast path")
	assert.Equal(t, args, got)
}
