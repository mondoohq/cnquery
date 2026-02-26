// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	privateca "cloud.google.com/go/security/privateca/apiv1"
	"cloud.google.com/go/security/privateca/apiv1/privatecapb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// extractCaPoolName extracts the CA pool name from a GCP Certificate Authority Service resource path.
// Path format: projects/{project}/locations/{location}/caPools/{caPoolName}/...
// Returns an empty string if the "caPools" segment is not found.
func extractCaPoolName(resourcePath string) string {
	parts := strings.Split(resourcePath, "/")
	for i, p := range parts {
		if p == "caPools" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func (g *mqlGcpProject) certificateAuthority() (*mqlGcpProjectCertificateAuthorityService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.certificateAuthorityService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectCertificateAuthorityService), nil
}

func initGcpProjectCertificateAuthorityService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectCertificateAuthorityService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/certificateAuthorityService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectCertificateAuthorityService) caPools() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(privateca.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := privateca.NewCertificateAuthorityClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCaPools(ctx, &privatecapb.ListCaPoolsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		pool, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		issuancePolicy, err := protoToDict(pool.IssuancePolicy)
		if err != nil {
			return nil, err
		}
		publishingOptions, err := protoToDict(pool.PublishingOptions)
		if err != nil {
			return nil, err
		}

		mqlPool, err := CreateResource(g.MqlRuntime, "gcp.project.certificateAuthorityService.caPool", map[string]*llx.RawData{
			"projectId":         llx.StringData(projectId),
			"resourcePath":      llx.StringData(pool.Name),
			"name":              llx.StringData(parseResourceName(pool.Name)),
			"location":          llx.StringData(parseLocationFromPath(pool.Name)),
			"tier":              llx.StringData(pool.Tier.String()),
			"issuancePolicy":    llx.DictData(issuancePolicy),
			"publishingOptions": llx.DictData(publishingOptions),
			"labels":            llx.MapData(convert.MapToInterfaceMap(pool.Labels), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPool)
	}

	return res, nil
}

func (g *mqlGcpProjectCertificateAuthorityServiceCaPool) id() (string, error) {
	if g.ResourcePath.Error != nil {
		return "", g.ResourcePath.Error
	}
	return g.ResourcePath.Data, nil
}

func (g *mqlGcpProjectCertificateAuthorityServiceCaPool) certificateAuthorities() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	projectId := g.ProjectId.Data
	poolPath := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(privateca.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := privateca.NewCertificateAuthorityClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificateAuthorities(ctx, &privatecapb.ListCertificateAuthoritiesRequest{
		Parent: poolPath,
	})

	var res []any
	for {
		ca, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		keySpec, err := protoToDict(ca.KeySpec)
		if err != nil {
			return nil, err
		}
		config, err := protoToDict(ca.Config)
		if err != nil {
			return nil, err
		}
		subordinateConfig, err := protoToDict(ca.SubordinateConfig)
		if err != nil {
			return nil, err
		}
		accessUrls, err := protoToDict(ca.AccessUrls)
		if err != nil {
			return nil, err
		}

		caPoolName := extractCaPoolName(ca.Name)

		var createdAt *llx.RawData
		if ca.CreateTime != nil {
			createdAt = llx.TimeData(ca.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if ca.UpdateTime != nil {
			updatedAt = llx.TimeData(ca.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		var deletedAt *llx.RawData
		if ca.DeleteTime != nil {
			deletedAt = llx.TimeData(ca.DeleteTime.AsTime())
		} else {
			deletedAt = llx.NilData
		}

		var expireTime *llx.RawData
		if ca.ExpireTime != nil {
			expireTime = llx.TimeData(ca.ExpireTime.AsTime())
		} else {
			expireTime = llx.NilData
		}

		mqlCa, err := CreateResource(g.MqlRuntime, "gcp.project.certificateAuthorityService.certificateAuthority", map[string]*llx.RawData{
			"projectId":         llx.StringData(projectId),
			"resourcePath":      llx.StringData(ca.Name),
			"name":              llx.StringData(parseResourceName(ca.Name)),
			"location":          llx.StringData(parseLocationFromPath(ca.Name)),
			"caPool":            llx.StringData(caPoolName),
			"type":              llx.StringData(ca.Type.String()),
			"state":             llx.StringData(ca.State.String()),
			"keySpec":           llx.DictData(keySpec),
			"config":            llx.DictData(config),
			"lifetime":          llx.StringData(ca.Lifetime.String()),
			"pemCaCertificates": llx.ArrayData(convert.SliceAnyToInterface(ca.PemCaCertificates), types.String),
			"subordinateConfig": llx.DictData(subordinateConfig),
			"labels":            llx.MapData(convert.MapToInterfaceMap(ca.Labels), types.String),
			"gcsBucket":         llx.StringData(ca.GcsBucket),
			"accessUrls":        llx.DictData(accessUrls),
			"createdAt":         createdAt,
			"updatedAt":         updatedAt,
			"deletedAt":         deletedAt,
			"expireTime":        expireTime,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCa)
	}

	return res, nil
}

func (g *mqlGcpProjectCertificateAuthorityServiceCertificateAuthority) id() (string, error) {
	if g.ResourcePath.Error != nil {
		return "", g.ResourcePath.Error
	}
	return g.ResourcePath.Data, nil
}

func (g *mqlGcpProjectCertificateAuthorityServiceCaPool) certificates() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.ResourcePath.Error != nil {
		return nil, g.ResourcePath.Error
	}
	projectId := g.ProjectId.Data
	poolPath := g.ResourcePath.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(privateca.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := privateca.NewCertificateAuthorityClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListCertificates(ctx, &privatecapb.ListCertificatesRequest{
		Parent: poolPath,
	})

	var res []any
	for {
		cert, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		subjectDescription, err := protoToDict(cert.CertificateDescription)
		if err != nil {
			return nil, err
		}
		certConfig, err := protoToDict(cert.GetConfig())
		if err != nil {
			return nil, err
		}
		revocationDetails, err := protoToDict(cert.RevocationDetails)
		if err != nil {
			return nil, err
		}

		caPoolName := extractCaPoolName(cert.Name)

		var createdAt *llx.RawData
		if cert.CreateTime != nil {
			createdAt = llx.TimeData(cert.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if cert.UpdateTime != nil {
			updatedAt = llx.TimeData(cert.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		mqlCert, err := CreateResource(g.MqlRuntime, "gcp.project.certificateAuthorityService.certificate", map[string]*llx.RawData{
			"projectId":                  llx.StringData(projectId),
			"resourcePath":               llx.StringData(cert.Name),
			"name":                       llx.StringData(parseResourceName(cert.Name)),
			"location":                   llx.StringData(parseLocationFromPath(cert.Name)),
			"caPool":                     llx.StringData(caPoolName),
			"issuerCertificateAuthority": llx.StringData(cert.IssuerCertificateAuthority),
			"lifetime":                   llx.StringData(cert.Lifetime.String()),
			"subjectDescription":         llx.DictData(subjectDescription),
			"certDescription":            llx.DictData(certConfig),
			"pemCertificate":             llx.StringData(cert.PemCertificate),
			"pemCertificateChain":        llx.ArrayData(convert.SliceAnyToInterface(cert.PemCertificateChain), types.String),
			"revocationDetails":          llx.DictData(revocationDetails),
			"labels":                     llx.MapData(convert.MapToInterfaceMap(cert.Labels), types.String),
			"createdAt":                  createdAt,
			"updatedAt":                  updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCert)
	}

	return res, nil
}

func (g *mqlGcpProjectCertificateAuthorityServiceCertificate) id() (string, error) {
	if g.ResourcePath.Error != nil {
		return "", g.ResourcePath.Error
	}
	return g.ResourcePath.Data, nil
}
