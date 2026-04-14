// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/iap/apiv1/iappb"
	"github.com/stretchr/testify/assert"
)

func TestIAPBrandFields(t *testing.T) {
	t.Run("brand proto fields", func(t *testing.T) {
		brand := &iappb.Brand{
			Name:             "projects/123/brands/456",
			ApplicationTitle: "My App",
			SupportEmail:     "support@example.com",
			OrgInternalOnly:  true,
		}
		assert.Equal(t, "projects/123/brands/456", brand.Name)
		assert.Equal(t, "My App", brand.ApplicationTitle)
		assert.Equal(t, "support@example.com", brand.SupportEmail)
		assert.True(t, brand.OrgInternalOnly)
	})
}

func TestIAPTunnelDestGroupFields(t *testing.T) {
	t.Run("tunnel dest group with cidrs and fqdns", func(t *testing.T) {
		group := &iappb.TunnelDestGroup{
			Name:  "projects/123/iap_tunnel/locations/us-central1/destGroups/my-group",
			Cidrs: []string{"10.0.0.0/8", "172.16.0.0/12"},
			Fqdns: []string{"host1.example.com", "host2.example.com"},
		}
		assert.Equal(t, 2, len(group.Cidrs))
		assert.Equal(t, 2, len(group.Fqdns))
		assert.Equal(t, "10.0.0.0/8", group.Cidrs[0])
		assert.Equal(t, "host1.example.com", group.Fqdns[0])
	})

	t.Run("empty cidrs and fqdns", func(t *testing.T) {
		group := &iappb.TunnelDestGroup{
			Name: "projects/123/iap_tunnel/locations/us-central1/destGroups/empty",
		}
		assert.Empty(t, group.Cidrs)
		assert.Empty(t, group.Fqdns)
	})
}
