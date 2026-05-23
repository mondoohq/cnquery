// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlCloudflareOneInternal struct {
	ZoneID    string
	AccountID string
}

// isSAMLIdpType returns true for IdP types that authenticate via SAML 2.0
// (either the generic SAML connector or vendor-specific SAML connectors).
// OIDC and social IdPs (e.g., google, github, yandex) are excluded.
//
// Note: Cloudflare's `okta` connector is OIDC. Okta deployments that use
// SAML come through as the generic `saml` type, which is already covered.
func isSAMLIdpType(t string) bool {
	switch t {
	case "saml", "adfs", "centrify", "onelogin", "ping", "pingone":
		return true
	}
	return false
}

func (c *mqlCloudflareZone) one() (*mqlCloudflareOne, error) {
	res, err := CreateResource(c.MqlRuntime, "cloudflare.one", map[string]*llx.RawData{
		"__id": llx.StringData("cloudflare.one@" + c.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	one := res.(*mqlCloudflareOne)
	one.ZoneID = c.Id.Data

	acc := c.GetAccount()
	if acc.Error != nil {
		return nil, acc.Error
	}
	one.AccountID = acc.Data.GetId().Data

	return one, nil
}

type mqlCloudflareOneAppInternal struct {
	// appPolicies caches the access policies embedded in the application
	// record so the typed policies() accessor needs no extra API call.
	appPolicies []cloudflare.AccessPolicy
}

func (c *mqlCloudflareOneApp) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

// accessRules normalizes an Access include/require/exclude rule list (a list of
// generic condition objects) into a dict-safe slice.
func accessRules(in []interface{}) []any {
	if in == nil {
		return []any{}
	}
	return in
}

// newMqlCloudflareOneAccessPolicy builds an access policy resource, including
// its include/require/exclude condition rules. It is shared by the account
// accessPolicies() listing and the per-application policies() accessor.
func newMqlCloudflareOneAccessPolicy(runtime *plugin.Runtime, p cloudflare.AccessPolicy) (plugin.Resource, error) {
	return NewResource(runtime, "cloudflare.one.accessPolicy", map[string]*llx.RawData{
		"id":         llx.StringData(p.ID),
		"name":       llx.StringData(p.Name),
		"decision":   llx.StringData(p.Decision),
		"precedence": llx.IntData(int64(p.Precedence)),
		"createdAt":  llx.TimeDataPtr(p.CreatedAt),
		"updatedAt":  llx.TimeDataPtr(p.UpdatedAt),
		"include":    llx.ArrayData(accessRules(p.Include), types.Dict),
		"require":    llx.ArrayData(accessRules(p.Require), types.Dict),
		"exclude":    llx.ArrayData(accessRules(p.Exclude), types.Dict),
	})
}

func (c *mqlCloudflareOneApp) policies() ([]any, error) {
	result := make([]any, 0, len(c.appPolicies))
	for i := range c.appPolicies {
		res, err := newMqlCloudflareOneAccessPolicy(c.MqlRuntime, c.appPolicies[i])
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (c *mqlCloudflareOne) apps() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}

	var result []any
	for {
		records, info, err := conn.Cf.ListAccessApplications(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.ZoneID,
			Level:      cloudflare.ZoneRouteLevel,
		}, cloudflare.ListAccessApplicationsParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			resourceData := map[string]*llx.RawData{
				"id":     llx.StringData(rec.ID),
				"aud":    llx.StringData(rec.AUD),
				"name":   llx.StringData(rec.Name),
				"domain": llx.StringData(rec.Domain),

				"allowedIdentityProviders": llx.ArrayData(convert.SliceAnyToInterface(rec.AllowedIdps), types.String),

				"appLauncherVisible":     llx.BoolDataPtr(rec.AppLauncherVisible),
				"autoRedirectToIdentity": llx.BoolDataPtr(rec.AutoRedirectToIdentity),
				"optionsPreflightBypass": llx.BoolDataPtr(rec.OptionsPreflightBypass),

				"customDenyMessage":      llx.StringData(rec.CustomDenyMessage),
				"customDenyUrl":          llx.StringData(rec.CustomDenyURL),
				"serviceAuth401Redirect": llx.BoolDataPtr(rec.ServiceAuth401Redirect),

				"enableBindingCookie":     llx.BoolDataPtr(rec.EnableBindingCookie),
				"httpOnlyCookieAttribute": llx.BoolDataPtr(rec.HttpOnlyCookieAttribute),
				"sameSiteCookieAttribute": llx.StringData(rec.SameSiteCookieAttribute),

				"logoUrl":          llx.StringData(rec.LogoURL),
				"sessionDuration":  llx.StringData(rec.SessionDuration),
				"skipInterstitial": llx.BoolDataPtr(rec.SkipInterstitial),

				"type": llx.StringData(string(rec.Type)),

				"createdAt": llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt": llx.TimeDataPtr(rec.UpdatedAt),

				"corsHeaders": llx.NilData,
			}

			if rec.CorsHeaders != nil {
				headers := rec.CorsHeaders
				corsHeaders, err := NewResource(c.MqlRuntime, "cloudflare.corsHeaders", map[string]*llx.RawData{
					"allowAllHeaders":  llx.BoolData(headers.AllowAllHeaders),
					"allowAllMethods":  llx.BoolData(headers.AllowAllMethods),
					"allowAllOrigins":  llx.BoolData(headers.AllowAllOrigins),
					"allowCredentials": llx.BoolData(headers.AllowCredentials),
					"allowedHeaders":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedHeaders), types.String),
					"allowedMethods":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedMethods), types.String),
					"allowedOrigins":   llx.ArrayData(convert.SliceAnyToInterface(headers.AllowedOrigins), types.String),
					"maxAge":           llx.IntData(headers.MaxAge),
				})
				if err != nil {
					return nil, err
				}
				resourceData["corsHeaders"] = llx.ResourceData(corsHeaders, corsHeaders.MqlName())
			}

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.app", resourceData)
			if err != nil {
				return nil, err
			}
			res.(*mqlCloudflareOneApp).appPolicies = rec.Policies

			result = append(result, res)

		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}

func (c *mqlCloudflareOneAccessPolicy) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) accessPolicies() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.ListAccessPolicies(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.AccountID,
			Level:      cloudflare.AccountRouteLevel,
		}, cloudflare.ListAccessPoliciesParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			res, err := newMqlCloudflareOneAccessPolicy(c.MqlRuntime, rec)
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}

func (c *mqlCloudflareOneAccessGroup) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) accessGroups() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.ListAccessGroups(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.AccountID,
			Level:      cloudflare.AccountRouteLevel,
		}, cloudflare.ListAccessGroupsParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.accessGroup", map[string]*llx.RawData{
				"id":        llx.StringData(rec.ID),
				"name":      llx.StringData(rec.Name),
				"createdAt": llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt": llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}

func (c *mqlCloudflareOneServiceToken) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

// serviceTokens lists Access service tokens. cloudflare-go's
// ListAccessServiceTokens does not paginate, so we call the endpoint directly
// via api.Raw and walk pages using the response's result_info block.
func (c *mqlCloudflareOne) serviceTokens() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	var (
		result  []any
		page    = 1
		perPage = 50
	)

	for {
		uri := fmt.Sprintf("/accounts/%s/access/service_tokens?page=%d&per_page=%d", c.AccountID, page, perPage)
		raw, err := conn.Cf.Raw(context.TODO(), http.MethodGet, uri, nil, nil)
		if err != nil {
			return nil, err
		}

		var records []cloudflare.AccessServiceToken
		if len(raw.Result) > 0 {
			if err := json.Unmarshal(raw.Result, &records); err != nil {
				return nil, fmt.Errorf("failed to decode access service tokens response: %w", err)
			}
		}

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.serviceToken", map[string]*llx.RawData{
				"id":         llx.StringData(rec.ID),
				"name":       llx.StringData(rec.Name),
				"clientId":   llx.StringData(rec.ClientID),
				"duration":   llx.StringData(rec.Duration),
				"expiresAt":  llx.TimeDataPtr(rec.ExpiresAt),
				"lastSeenAt": llx.TimeDataPtr(rec.LastSeenAt),
				"createdAt":  llx.TimeDataPtr(rec.CreatedAt),
				"updatedAt":  llx.TimeDataPtr(rec.UpdatedAt),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if raw.ResultInfo == nil || !raw.ResultInfo.HasMorePages() {
			break
		}
		page++
	}

	return result, nil
}

func (c *mqlCloudflareOne) organization() (*mqlCloudflareOneOrganization, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	org, _, err := conn.Cf.GetAccessOrganization(context.TODO(), &cloudflare.ResourceContainer{
		Identifier: c.AccountID,
		Level:      cloudflare.AccountRouteLevel,
	}, cloudflare.GetAccessOrganizationParams{})
	if err != nil {
		var notFound *cloudflare.NotFoundError
		var authN *cloudflare.AuthenticationError
		var authZ *cloudflare.AuthorizationError
		if errors.As(err, &notFound) || errors.As(err, &authN) || errors.As(err, &authZ) {
			c.Organization.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	res, err := NewResource(c.MqlRuntime, "cloudflare.one.organization", map[string]*llx.RawData{
		"__id":                           llx.StringData("cloudflare.one.organization@" + c.AccountID),
		"name":                           llx.StringData(org.Name),
		"authDomain":                     llx.StringData(org.AuthDomain),
		"isUiReadOnly":                   llx.BoolDataPtr(org.IsUIReadOnly),
		"userSeatExpirationInactiveTime": llx.StringData(org.UserSeatExpirationInactiveTime),
		"autoRedirectToIdentity":         llx.BoolDataPtr(org.AutoRedirectToIdentity),
		"sessionDuration":                llx.StringDataPtr(org.SessionDuration),
		"warpAuthSessionDuration":        llx.StringDataPtr(org.WarpAuthSessionDuration),
		"allowAuthenticateViaWarp":       llx.BoolDataPtr(org.AllowAuthenticateViaWarp),
		"createdAt":                      llx.TimeDataPtr(org.CreatedAt),
		"updatedAt":                      llx.TimeDataPtr(org.UpdatedAt),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlCloudflareOneOrganization), nil
}

func (c *mqlCloudflareOneIdp) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareOne) identityProviders() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.CloudflareConnection)

	cursor := &cloudflare.ResultInfo{}
	var result []any
	for {
		records, info, err := conn.Cf.ListAccessIdentityProviders(context.TODO(), &cloudflare.ResourceContainer{
			Identifier: c.ZoneID,
			Level:      cloudflare.ZoneRouteLevel,
		}, cloudflare.ListAccessIdentityProvidersParams{
			ResultInfo: *cursor,
		})
		if err != nil {
			return nil, err
		}

		cursor = info

		for i := range records {
			rec := records[i]

			res, err := NewResource(c.MqlRuntime, "cloudflare.one.idp", map[string]*llx.RawData{
				"id":                 llx.StringData(rec.ID),
				"name":               llx.StringData(rec.Name),
				"type":               llx.StringData(string(rec.Type)),
				"saml":               llx.BoolData(isSAMLIdpType(string(rec.Type))),
				"ssoTargetUrl":       llx.StringData(rec.Config.SsoTargetURL),
				"issuerUrl":          llx.StringData(rec.Config.IssuerURL),
				"signRequest":        llx.BoolData(rec.Config.SignRequest),
				"idpPublicCert":      llx.StringData(rec.Config.IdpPublicCert),
				"emailAttributeName": llx.StringData(rec.Config.EmailAttributeName),
				"attributes":         llx.ArrayData(convert.SliceAnyToInterface(rec.Config.Attributes), types.String),
				"scimEnabled":        llx.BoolData(rec.ScimConfig.Enabled),
			})
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		if !cursor.HasMorePages() {
			break
		}
	}

	return result, nil
}
