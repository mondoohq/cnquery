// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	accesscontextmanager "cloud.google.com/go/accesscontextmanager/apiv1"
	acmpb "cloud.google.com/go/accesscontextmanager/apiv1/accesscontextmanagerpb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpAccesscontextmanagerAccessPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpAccesscontextmanagerAccessLevel) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpAccesscontextmanagerServicePerimeter) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func newACMClient(conn *connection.GcpConnection) (*accesscontextmanager.Client, error) {
	creds, err := conn.Credentials(accesscontextmanager.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	return accesscontextmanager.NewClient(context.Background(), option.WithCredentials(creds))
}

func (g *mqlGcpOrganization) accessPolicies() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	orgId, err := conn.OrganizationID()
	if err != nil {
		return nil, err
	}

	client, err := newACMClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListAccessPolicies(context.Background(), &acmpb.ListAccessPoliciesRequest{
		Parent: "organizations/" + orgId,
	})

	var res []any
	for {
		policy, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.accesscontextmanager.accessPolicy", map[string]*llx.RawData{
			"name":   llx.StringData(policy.Name),
			"title":  llx.StringData(policy.Title),
			"parent": llx.StringData(policy.Parent),
			"etag":   llx.StringData(policy.Etag),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}

	return res, nil
}

func (g *mqlGcpAccesscontextmanagerAccessPolicy) accessLevels() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	policyName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := newACMClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListAccessLevels(context.Background(), &acmpb.ListAccessLevelsRequest{
		Parent: policyName,
	})

	var res []any
	for {
		level, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		basic, err := protoToDict(level.GetBasic())
		if err != nil {
			return nil, err
		}
		custom, err := protoToDict(level.GetCustom())
		if err != nil {
			return nil, err
		}

		mqlLevel, err := CreateResource(g.MqlRuntime, "gcp.accesscontextmanager.accessLevel", map[string]*llx.RawData{
			"name":        llx.StringData(level.Name),
			"title":       llx.StringData(level.Title),
			"description": llx.StringData(level.Description),
			"basic":       llx.DictData(basic),
			"custom":      llx.DictData(custom),
			"createTime":  llx.TimeDataPtr(timestampAsTimePtr(level.CreateTime)),
			"updateTime":  llx.TimeDataPtr(timestampAsTimePtr(level.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlLevel)
	}

	return res, nil
}

func (g *mqlGcpAccesscontextmanagerAccessPolicy) servicePerimeters() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	policyName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := newACMClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListServicePerimeters(context.Background(), &acmpb.ListServicePerimetersRequest{
		Parent: policyName,
	})

	var res []any
	for {
		perimeter, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		status, err := protoToDict(perimeter.Status)
		if err != nil {
			return nil, err
		}
		spec, err := protoToDict(perimeter.Spec)
		if err != nil {
			return nil, err
		}

		mqlPerimeter, err := CreateResource(g.MqlRuntime, "gcp.accesscontextmanager.servicePerimeter", map[string]*llx.RawData{
			"name":                  llx.StringData(perimeter.Name),
			"title":                 llx.StringData(perimeter.Title),
			"description":           llx.StringData(perimeter.Description),
			"perimeterType":         llx.StringData(perimeter.PerimeterType.String()),
			"status":                llx.DictData(status),
			"spec":                  llx.DictData(spec),
			"useExplicitDryRunSpec": llx.BoolData(perimeter.UseExplicitDryRunSpec),
			"createTime":            llx.TimeDataPtr(timestampAsTimePtr(perimeter.CreateTime)),
			"updateTime":            llx.TimeDataPtr(timestampAsTimePtr(perimeter.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPerimeter)
	}

	return res, nil
}
