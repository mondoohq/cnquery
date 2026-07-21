// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganization) workforcePools() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	orgId, err := conn.OrganizationID()
	if err != nil {
		return nil, err
	}

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iamSvc, err := iam.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var pools []any
	err = iamSvc.Locations.WorkforcePools.List("locations/global").
		Parent("organizations/"+orgId).
		ShowDeleted(true).
		Pages(ctx, func(resp *iam.ListWorkforcePoolsResponse) error {
			for _, p := range resp.WorkforcePools {
				var disableProgrammaticSignin bool
				allowedServices := []any{}
				if p.AccessRestrictions != nil {
					disableProgrammaticSignin = p.AccessRestrictions.DisableProgrammaticSignin
					for _, s := range p.AccessRestrictions.AllowedServices {
						if s != nil && s.Domain != "" {
							allowedServices = append(allowedServices, s.Domain)
						}
					}
				}

				mqlPool, err := CreateResource(g.MqlRuntime, "gcp.organization.workforcePool",
					map[string]*llx.RawData{
						"name":                      llx.StringData(p.Name),
						"poolId":                    llx.StringData(lastSegment(p.Name)),
						"parent":                    llx.StringData(p.Parent),
						"displayName":               llx.StringData(p.DisplayName),
						"description":               llx.StringData(p.Description),
						"state":                     llx.StringData(p.State),
						"disabled":                  llx.BoolData(p.Disabled),
						"sessionDuration":           llx.StringData(p.SessionDuration),
						"expireTime":                llx.TimeDataPtr(parseTime(p.ExpireTime)),
						"disableProgrammaticSignin": llx.BoolData(disableProgrammaticSignin),
						"allowedServices":           llx.ArrayData(allowedServices, types.String),
					})
				if err != nil {
					return err
				}
				pools = append(pools, mqlPool)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (g *mqlGcpOrganizationWorkforcePool) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpOrganizationWorkforcePool) providers() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	poolName := g.Name.Data
	if poolName == "" {
		return nil, errors.New("workforce pool has no name")
	}
	if g.State.Error != nil {
		return nil, g.State.Error
	}
	// A deleted pool 404s on its providers list; skip the call.
	if g.State.Data != "" && g.State.Data != "ACTIVE" {
		g.Providers.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iamSvc, err := iam.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var providers []any
	err = iamSvc.Locations.WorkforcePools.Providers.List(poolName).
		ShowDeleted(true).
		Pages(ctx, func(resp *iam.ListWorkforcePoolProvidersResponse) error {
			for _, p := range resp.WorkforcePoolProviders {
				providerType, oidcIssuer, oidcClientId, samlMetadata := flattenWorkforceProviderConfig(p)

				var extraAttributesType, extraAttributesIssuerUri string
				if p.ExtraAttributesOauth2Client != nil {
					extraAttributesType = p.ExtraAttributesOauth2Client.AttributesType
					extraAttributesIssuerUri = p.ExtraAttributesOauth2Client.IssuerUri
				}

				mqlProvider, err := CreateResource(g.MqlRuntime, "gcp.organization.workforcePool.provider",
					map[string]*llx.RawData{
						"name":                     llx.StringData(p.Name),
						"providerId":               llx.StringData(lastSegment(p.Name)),
						"poolId":                   llx.StringData(lastSegment(poolName)),
						"displayName":              llx.StringData(p.DisplayName),
						"description":              llx.StringData(p.Description),
						"state":                    llx.StringData(p.State),
						"disabled":                 llx.BoolData(p.Disabled),
						"expireTime":               llx.TimeDataPtr(parseTime(p.ExpireTime)),
						"attributeMapping":         llx.MapData(convert.MapToInterfaceMap(p.AttributeMapping), types.String),
						"attributeCondition":       llx.StringData(p.AttributeCondition),
						"detailedAuditLogging":     llx.BoolData(p.DetailedAuditLogging),
						"scimUsage":                llx.StringData(p.ScimUsage),
						"providerType":             llx.StringData(providerType),
						"oidcIssuerUri":            llx.StringData(oidcIssuer),
						"oidcClientId":             llx.StringData(oidcClientId),
						"samlIdpMetadataXml":       llx.StringData(samlMetadata),
						"extraAttributesType":      llx.StringData(extraAttributesType),
						"extraAttributesIssuerUri": llx.StringData(extraAttributesIssuerUri),
					})
				if err != nil {
					return err
				}
				providers = append(providers, mqlProvider)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (g *mqlGcpOrganizationWorkforcePoolProvider) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// flattenWorkforceProviderConfig extracts the protocol discriminator and the
// per-protocol trust fields from a WorkforcePoolProvider. Exactly one of Oidc
// or Saml should be set; the other returns zero values.
func flattenWorkforceProviderConfig(p *iam.WorkforcePoolProvider) (providerType, oidcIssuer, oidcClientId, samlMetadata string) {
	switch {
	case p.Oidc != nil:
		providerType = "oidc"
		oidcIssuer = p.Oidc.IssuerUri
		oidcClientId = p.Oidc.ClientId
	case p.Saml != nil:
		providerType = "saml"
		samlMetadata = p.Saml.IdpMetadataXml
	}
	return
}
