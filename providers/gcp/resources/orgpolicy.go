// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	orgpolicy "cloud.google.com/go/orgpolicy/apiv2"
	"cloud.google.com/go/orgpolicy/apiv2/orgpolicypb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrgPolicy) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

// listOrgPolicies fetches org policies for a given parent resource.
// parentResourceName should be "organizations/{id}" or "projects/{id}".
func listOrgPolicies(runtime *plugin.Runtime, conn *connection.GcpConnection, parentResourceName string) ([]any, error) {
	creds, err := conn.Credentials(orgpolicy.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := orgpolicy.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListPolicies(ctx, &orgpolicypb.ListPoliciesRequest{
		Parent: parentResourceName,
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

		spec, err := protoToDict(policy.Spec)
		if err != nil {
			return nil, err
		}
		dryRunSpec, err := protoToDict(policy.DryRunSpec)
		if err != nil {
			return nil, err
		}

		constraintName := extractConstraintName(policy.Name)

		var updatedAt *llx.RawData
		if policy.Spec != nil && policy.Spec.UpdateTime != nil {
			updatedAt = llx.TimeData(policy.Spec.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		mqlPolicy, err := CreateResource(runtime, "gcp.orgPolicy", map[string]*llx.RawData{
			"id":             llx.StringData(policy.Name),
			"name":           llx.StringData(policy.Name),
			"constraintName": llx.StringData(constraintName),
			"spec":           llx.DictData(spec),
			"dryRunSpec":     llx.DictData(dryRunSpec),
			"etag":           llx.StringData(policy.Etag),
			"updatedAt":      updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPolicy)
	}

	return res, nil
}

func (g *mqlGcpOrganization) orgPolicies() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	orgId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	// orgId is already in "organizations/{id}" format from initGcpOrganization
	return listOrgPolicies(g.MqlRuntime, conn, orgId)
}

// extractConstraintName extracts the constraint name from a full org policy resource path.
// Format: {parent}/policies/{constraintName}
// Returns the full name unchanged if the "/policies/" segment is not found.
func extractConstraintName(policyName string) string {
	if idx := strings.LastIndex(policyName, "/policies/"); idx != -1 {
		return policyName[idx+len("/policies/"):]
	}
	return policyName
}

func (g *mqlGcpProject) orgPolicies() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	return listOrgPolicies(g.MqlRuntime, conn, "projects/"+projectId)
}

func (g *mqlGcpOrgPolicyConstraint) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProject) orgPolicyConstraints() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(orgpolicy.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := orgpolicy.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var res []any
	it := client.ListConstraints(ctx, &orgpolicypb.ListConstraintsRequest{
		Parent: "projects/" + projectId,
	})
	for {
		c, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		listConstraint, err := protoToDict(c.GetListConstraint())
		if err != nil {
			return nil, err
		}
		booleanConstraint, err := protoToDict(c.GetBooleanConstraint())
		if err != nil {
			return nil, err
		}

		mqlConstraint, err := CreateResource(g.MqlRuntime, "gcp.orgPolicy.constraint", map[string]*llx.RawData{
			"name":              llx.StringData(c.Name),
			"displayName":       llx.StringData(c.DisplayName),
			"description":       llx.StringData(c.Description),
			"constraintDefault": llx.StringData(c.ConstraintDefault.String()),
			"listConstraint":    llx.DictData(listConstraint),
			"booleanConstraint": llx.DictData(booleanConstraint),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlConstraint)
	}
	return res, nil
}

func (g *mqlGcpOrgPolicyCustomConstraint) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpOrganization) customConstraints() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	// orgId is already in "organizations/{id}" format from initGcpOrganization
	orgId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(orgpolicy.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := orgpolicy.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var res []any
	it := client.ListCustomConstraints(ctx, &orgpolicypb.ListCustomConstraintsRequest{
		Parent: orgId,
	})
	for {
		c, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		methodTypes := make([]any, 0, len(c.MethodTypes))
		for _, m := range c.MethodTypes {
			methodTypes = append(methodTypes, m.String())
		}

		mqlConstraint, err := CreateResource(g.MqlRuntime, "gcp.orgPolicy.customConstraint", map[string]*llx.RawData{
			"name":          llx.StringData(c.Name),
			"displayName":   llx.StringData(c.DisplayName),
			"description":   llx.StringData(c.Description),
			"resourceTypes": llx.ArrayData(convert.SliceAnyToInterface(c.ResourceTypes), types.String),
			"methodTypes":   llx.ArrayData(methodTypes, types.String),
			"condition":     llx.StringData(c.Condition),
			"actionType":    llx.StringData(c.ActionType.String()),
			"updated":       llx.TimeDataPtr(timestampAsTimePtr(c.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlConstraint)
	}
	return res, nil
}
