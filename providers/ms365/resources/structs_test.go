// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/stretchr/testify/assert"
)

// Guards against the inverted nil-check the function used to carry —
// it now returns nil on a nil input instead of panicking on the
// dereference inside the constructor.
func TestNewAdminConsentRequestPolicy_NilInput(t *testing.T) {
	assert.Nil(t, newAdminConsentRequestPolicy(nil))
}

func TestDirectoryObjectDisplayName(t *testing.T) {
	name := "Alice Admin"
	tests := []struct {
		name string
		in   map[string]any
		want string
	}{
		{"nil map", nil, ""},
		{"missing key", map[string]any{"other": "x"}, ""},
		{"string value", map[string]any{"displayName": "Global Admins"}, "Global Admins"},
		{"pointer value", map[string]any{"displayName": &name}, name},
		{"nil pointer", map[string]any{"displayName": (*string)(nil)}, ""},
		{"wrong type", map[string]any{"displayName": 42}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, directoryObjectDisplayName(tc.in))
		})
	}
}

func TestDirectoryPrincipalInfo(t *testing.T) {
	t.Run("nil principal", func(t *testing.T) {
		pt, pn := directoryPrincipalInfo(nil)
		assert.Empty(t, pt)
		assert.Empty(t, pn)
	})

	t.Run("user with display name", func(t *testing.T) {
		p := models.NewDirectoryObject()
		p.SetOdataType(ptr("#microsoft.graph.user"))
		p.SetAdditionalData(map[string]any{"displayName": "Alice Admin"})
		pt, pn := directoryPrincipalInfo(p)
		assert.Equal(t, "user", pt)
		assert.Equal(t, "Alice Admin", pn)
	})

	t.Run("group without display name", func(t *testing.T) {
		p := models.NewDirectoryObject()
		p.SetOdataType(ptr("#microsoft.graph.group"))
		pt, pn := directoryPrincipalInfo(p)
		assert.Equal(t, "group", pt)
		assert.Empty(t, pn)
	})

	t.Run("missing odata type", func(t *testing.T) {
		p := models.NewDirectoryObject()
		pt, pn := directoryPrincipalInfo(p)
		assert.Empty(t, pt)
		assert.Empty(t, pn)
	})
}
