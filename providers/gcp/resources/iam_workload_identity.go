// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// wifPoolHostsProviders reports whether a pool's Mode lets it host
// user-defined providers. Per the GCP IAM API, an unspecified mode is treated
// as FEDERATION_ONLY; TRUST_DOMAIN and SYSTEM_TRUST_DOMAIN pools (e.g. GKE's
// `*.svc.id.goog` pool) reject ListWorkloadIdentityPoolProviders.
func wifPoolHostsProviders(mode string) bool {
	return mode == "" || mode == "MODE_UNSPECIFIED" || mode == "FEDERATION_ONLY"
}

func (g *mqlGcpProjectIamService) workloadIdentityPools() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

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

	parent := fmt.Sprintf("projects/%s/locations/global", projectId)
	var pools []any
	err = iamSvc.Projects.Locations.WorkloadIdentityPools.List(parent).
		ShowDeleted(true).
		Pages(ctx, func(resp *iam.ListWorkloadIdentityPoolsResponse) error {
			for _, p := range resp.WorkloadIdentityPools {
				mqlPool, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.workloadIdentityPool",
					map[string]*llx.RawData{
						"projectId":   llx.StringData(projectId),
						"name":        llx.StringData(p.Name),
						"poolId":      llx.StringData(lastSegment(p.Name)),
						"displayName": llx.StringData(p.DisplayName),
						"description": llx.StringData(p.Description),
						"state":       llx.StringData(p.State),
						"disabled":    llx.BoolData(p.Disabled),
						"expireTime":  llx.TimeDataPtr(parseTime(p.ExpireTime)),
						"mode":        llx.StringData(p.Mode),
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

func (g *mqlGcpProjectIamServiceWorkloadIdentityPool) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectIamServiceWorkloadIdentityPool) providers() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	poolName := g.Name.Data
	if poolName == "" {
		return nil, errors.New("workload identity pool has no name")
	}
	if g.Mode.Error != nil {
		return nil, g.Mode.Error
	}
	if g.State.Error != nil {
		return nil, g.State.Error
	}
	// Skip the list call for pools the API will reject anyway:
	// - non-ACTIVE pools (DELETED pools 404 on the providers list);
	// - TRUST_DOMAIN / SYSTEM_TRUST_DOMAIN pools (e.g. GKE's `*.svc.id.goog`
	//   pool), which 400 with "RPC Method ... is not supported on resource".
	if g.State.Data != "" && g.State.Data != "ACTIVE" {
		g.Providers.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	if !wifPoolHostsProviders(g.Mode.Data) {
		g.Providers.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.PoolId.Error != nil {
		return nil, g.PoolId.Error
	}
	poolId := g.PoolId.Data

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
	err = iamSvc.Projects.Locations.WorkloadIdentityPools.Providers.List(poolName).
		ShowDeleted(true).
		Pages(ctx, func(resp *iam.ListWorkloadIdentityPoolProvidersResponse) error {
			for _, p := range resp.WorkloadIdentityPoolProviders {
				providerType, awsAccountId, oidcIssuer, oidcAudiences, samlMetadata := flattenWifProviderConfig(p)

				mqlProvider, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.workloadIdentityPool.provider",
					map[string]*llx.RawData{
						"projectId":            llx.StringData(projectId),
						"name":                 llx.StringData(p.Name),
						"providerId":           llx.StringData(lastSegment(p.Name)),
						"poolId":               llx.StringData(poolId),
						"displayName":          llx.StringData(p.DisplayName),
						"description":          llx.StringData(p.Description),
						"state":                llx.StringData(p.State),
						"disabled":             llx.BoolData(p.Disabled),
						"expireTime":           llx.TimeDataPtr(parseTime(p.ExpireTime)),
						"attributeMapping":     llx.MapData(convert.MapToInterfaceMap(p.AttributeMapping), types.String),
						"attributeCondition":   llx.StringData(p.AttributeCondition),
						"providerType":         llx.StringData(providerType),
						"awsAccountId":         llx.StringData(awsAccountId),
						"oidcIssuerUri":        llx.StringData(oidcIssuer),
						"oidcAllowedAudiences": llx.ArrayData(oidcAudiences, types.String),
						"samlIdpMetadataXml":   llx.StringData(samlMetadata),
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

func (g *mqlGcpProjectIamServiceWorkloadIdentityPoolProvider) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// flattenWifProviderConfig extracts the credential-family discriminator and
// the per-family fields from a WorkloadIdentityPoolProvider. Exactly one of
// Aws, Oidc, Saml, or X509 should be set; the rest return zero values. X.509
// trust-store details aren't surfaced — providerType=="x509" is enough for
// audits to flag the family, and the trust-store schema is left for a
// follow-up if there's demand.
func flattenWifProviderConfig(p *iam.WorkloadIdentityPoolProvider) (providerType, awsAccountId, oidcIssuer string, oidcAudiences []any, samlMetadata string) {
	oidcAudiences = []any{}
	switch {
	case p.Aws != nil:
		providerType = "aws"
		awsAccountId = p.Aws.AccountId
	case p.Oidc != nil:
		providerType = "oidc"
		oidcIssuer = p.Oidc.IssuerUri
		for _, a := range p.Oidc.AllowedAudiences {
			oidcAudiences = append(oidcAudiences, a)
		}
	case p.Saml != nil:
		providerType = "saml"
		samlMetadata = p.Saml.IdpMetadataXml
	case p.X509 != nil:
		providerType = "x509"
	}
	return
}

// lastSegment returns the substring after the final "/" in a slash-delimited
// resource name, or the original string if no "/" is present.
func lastSegment(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}
