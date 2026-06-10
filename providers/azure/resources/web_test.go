// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"
	"github.com/stretchr/testify/assert"
)

func TestSiteConfigCorsAllowedOrigins(t *testing.T) {
	t.Run("nil props returns empty slice", func(t *testing.T) {
		assert.Equal(t, []any{}, siteConfigCorsAllowedOrigins(nil))
	})
	t.Run("nil cors returns empty slice", func(t *testing.T) {
		assert.Equal(t, []any{}, siteConfigCorsAllowedOrigins(&web.SiteConfig{}))
	})
	t.Run("allowed origins are returned", func(t *testing.T) {
		a, b := "https://example.com", "*"
		props := &web.SiteConfig{Cors: &web.CorsSettings{AllowedOrigins: []*string{&a, &b}}}
		assert.Equal(t, []any{"https://example.com", "*"}, siteConfigCorsAllowedOrigins(props))
	})
}

func TestParseVersion(t *testing.T) {
	assert.False(t, isPlatformEol("python", ""))
	assert.False(t, isPlatformEol("python", "3.7"))
	assert.False(t, isPlatformEol("node", "12-lts"))
	assert.False(t, isPlatformEol("node", "10-lts"))
	assert.True(t, isPlatformEol("node", "11.1"))
	assert.True(t, isPlatformEol("node", "6.1"))
}
