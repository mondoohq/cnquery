// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"
)

// TestAuthSettingsV2Args pins the flattening of the V2 auth settings response.
//
// Each answer this resource reports comes from a different nested group
// (platform, globalValidation, httpSettings, login.tokenStore,
// identityProviders.azureActiveDirectory.validation), and reading one from the
// wrong place yields a zero value that reads as "not required" -- the same
// false negative the V1-only read produced, just one level down.
func TestAuthSettingsV2Args(t *testing.T) {
	full := web.SiteAuthSettingsV2{
		ID:   to.Ptr("/subscriptions/s/resourceGroups/rg/providers/Microsoft.Web/sites/app/config/authsettingsV2"),
		Name: to.Ptr("authsettingsV2"),
		Kind: to.Ptr("app"),
		Type: to.Ptr("Microsoft.Web/sites/config"),
		Properties: &web.SiteAuthSettingsV2Properties{
			Platform: &web.AuthPlatform{
				Enabled:        to.Ptr(true),
				RuntimeVersion: to.Ptr("~1"),
			},
			GlobalValidation: &web.GlobalValidation{
				RequireAuthentication:       to.Ptr(true),
				UnauthenticatedClientAction: to.Ptr(web.UnauthenticatedClientActionV2Return403),
				RedirectToProvider:          to.Ptr("azureactivedirectory"),
				ExcludedPaths:               []*string{to.Ptr("/health"), to.Ptr("/metrics")},
			},
			HTTPSettings: &web.HTTPSettings{
				RequireHTTPS: to.Ptr(true),
			},
			Login: &web.Login{
				TokenStore: &web.TokenStore{Enabled: to.Ptr(true)},
			},
			IdentityProviders: &web.IdentityProviders{
				AzureActiveDirectory: &web.AzureActiveDirectory{
					Enabled: to.Ptr(true),
					Validation: &web.AzureActiveDirectoryValidation{
						AllowedAudiences: []*string{to.Ptr("api://00000000-0000-0000-0000-000000000000")},
						DefaultAuthorizationPolicy: &web.DefaultAuthorizationPolicy{
							AllowedApplications: []*string{
								to.Ptr("11111111-1111-1111-1111-111111111111"),
								to.Ptr("22222222-2222-2222-2222-222222222222"),
							},
						},
					},
				},
			},
		},
	}

	t.Run("maps every nested group", func(t *testing.T) {
		args := authSettingsV2Args(full, nil)

		assert.Equal(t, args["id"].Value, args["__id"].Value)
		assert.Equal(t, "authsettingsV2", args["name"].Value)
		assert.Equal(t, "app", args["kind"].Value)
		assert.Equal(t, "Microsoft.Web/sites/config", args["type"].Value)

		assert.Equal(t, true, args["enabled"].Value, "platform.enabled")
		assert.Equal(t, "~1", args["runtimeVersion"].Value)
		assert.Equal(t, true, args["requireAuthentication"].Value, "globalValidation.requireAuthentication")
		assert.Equal(t, "Return403", args["unauthenticatedClientAction"].Value)
		assert.Equal(t, "azureactivedirectory", args["redirectToProvider"].Value)
		assert.Equal(t, []any{"/health", "/metrics"}, args["excludedPaths"].Value)
		assert.Equal(t, true, args["requireHttps"].Value, "httpSettings.requireHttps")
		assert.Equal(t, true, args["tokenStoreEnabled"].Value, "login.tokenStore.enabled")
		assert.Equal(t, true, args["azureActiveDirectoryEnabled"].Value)
		assert.Equal(t, []any{
			"11111111-1111-1111-1111-111111111111",
			"22222222-2222-2222-2222-222222222222",
		}, args["allowedApplications"].Value)
		assert.Equal(t, []any{"api://00000000-0000-0000-0000-000000000000"}, args["allowedAudiences"].Value)
	})

	// An app with authentication switched off returns the same shape with the
	// booleans false. These must not read as true, and the allowlists must be
	// empty rather than carrying over.
	t.Run("an unauthenticated app reads as unauthenticated", func(t *testing.T) {
		args := authSettingsV2Args(web.SiteAuthSettingsV2{
			ID: to.Ptr("/subscriptions/s/.../config/authsettingsV2"),
			Properties: &web.SiteAuthSettingsV2Properties{
				Platform: &web.AuthPlatform{Enabled: to.Ptr(false)},
				GlobalValidation: &web.GlobalValidation{
					RequireAuthentication:       to.Ptr(false),
					UnauthenticatedClientAction: to.Ptr(web.UnauthenticatedClientActionV2AllowAnonymous),
				},
			},
		}, nil)

		assert.Equal(t, false, args["enabled"].Value)
		assert.Equal(t, false, args["requireAuthentication"].Value)
		assert.Equal(t, "AllowAnonymous", args["unauthenticatedClientAction"].Value)
		assert.Empty(t, args["allowedApplications"].Value)
		assert.Empty(t, args["allowedAudiences"].Value)
	})

	// Every level of the V2 shape is a nullable pointer. An absent group must
	// leave its fields at the safe reading, not panic and not report a
	// protection that is not configured.
	t.Run("absent groups do not panic and stay at the safe reading", func(t *testing.T) {
		for name, settings := range map[string]web.SiteAuthSettingsV2{
			"nil properties":   {ID: to.Ptr("/a")},
			"empty properties": {ID: to.Ptr("/b"), Properties: &web.SiteAuthSettingsV2Properties{}},
			"login without store": {ID: to.Ptr("/c"), Properties: &web.SiteAuthSettingsV2Properties{
				Login: &web.Login{},
			}},
			"aad without validation": {ID: to.Ptr("/d"), Properties: &web.SiteAuthSettingsV2Properties{
				IdentityProviders: &web.IdentityProviders{
					AzureActiveDirectory: &web.AzureActiveDirectory{Enabled: to.Ptr(true)},
				},
			}},
			"validation without policy": {ID: to.Ptr("/e"), Properties: &web.SiteAuthSettingsV2Properties{
				IdentityProviders: &web.IdentityProviders{
					AzureActiveDirectory: &web.AzureActiveDirectory{
						Validation: &web.AzureActiveDirectoryValidation{},
					},
				},
			}},
		} {
			t.Run(name, func(t *testing.T) {
				args := authSettingsV2Args(settings, nil)
				require.NotNil(t, args)
				assert.Equal(t, false, args["requireAuthentication"].Value)
				assert.Equal(t, false, args["requireHttps"].Value)
				assert.Equal(t, false, args["tokenStoreEnabled"].Value)
				assert.Empty(t, args["unauthenticatedClientAction"].Value)
				assert.Empty(t, args["allowedApplications"].Value)
			})
		}
	})

	// A nil *bool must decode to false, never to the value of whatever the
	// previous record held.
	t.Run("absent booleans decode to false", func(t *testing.T) {
		args := authSettingsV2Args(web.SiteAuthSettingsV2{
			ID: to.Ptr("/f"),
			Properties: &web.SiteAuthSettingsV2Properties{
				GlobalValidation: &web.GlobalValidation{},
				HTTPSettings:     &web.HTTPSettings{},
				Login:            &web.Login{TokenStore: &web.TokenStore{}},
			},
		}, nil)
		assert.Equal(t, false, args["requireAuthentication"].Value)
		assert.Equal(t, false, args["requireHttps"].Value)
		assert.Equal(t, false, args["tokenStoreEnabled"].Value)
	})

	// The resource has to be creatable from these args: an argument key the
	// schema does not declare is rejected at creation, which is how a renamed
	// field shows up.
	t.Run("args create the resource", func(t *testing.T) {
		res, err := CreateResource(cacheIDTestRuntime(),
			"azure.subscription.webService.appsiteauthsettingsv2", authSettingsV2Args(full, nil))
		require.NoError(t, err)

		v2 := res.(*mqlAzureSubscriptionWebServiceAppsiteauthsettingsv2)
		assert.True(t, v2.RequireAuthentication.Data)
		assert.Equal(t, "Return403", v2.UnauthenticatedClientAction.Data)
		assert.True(t, v2.RequireHttps.Data)
		assert.True(t, v2.TokenStoreEnabled.Data)
	})
}
