// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strconv"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganization) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func initGcpOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	// determine org from project in transport
	orgId, err := conn.OrganizationID()
	if err != nil {
		log.Error().Err(err).Msg("could not determine organization id")
		return nil, nil, err
	}

	name := "organizations/" + orgId
	org, err := svc.Organizations.Get(name).Do()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(org.Name)
	args["name"] = llx.StringData(org.DisplayName)
	args["state"] = llx.StringData(org.State)
	args["lifecycleState"] = llx.StringData(org.State)

	return args, nil, nil
}

func (g *mqlGcpOrganization) name() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpOrganization) state() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpOrganization) lifecycleState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpOrganization) iamPolicy() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// determine org from project in transport
	orgId, err := conn.OrganizationID()
	if err != nil {
		return nil, err
	}

	name := "organizations/" + orgId
	orgpolicy, err := svc.Organizations.GetIamPolicy(name, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range orgpolicy.Bindings {
		b := orgpolicy.Bindings[i]

		mqlServiceaccount, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":      llx.StringData(name + "-" + strconv.Itoa(i)),
			"role":    llx.StringData(b.Role),
			"members": llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpOrganization) folders() (*mqlGcpFolders, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	orgId := g.Id.Data
	res, err := CreateResource(g.MqlRuntime, "gcp.folders", map[string]*llx.RawData{
		"parentId": llx.StringData(orgId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpFolders), nil
}

func (g *mqlGcpOrganization) projects() (*mqlGcpProjects, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	orgId := g.Id.Data
	res, err := CreateResource(g.MqlRuntime, "gcp.projects", map[string]*llx.RawData{
		"parentId": llx.StringData(orgId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjects), nil
}

func (g *mqlGcpResourcemanagerBinding) id() (string, error) {
	return g.Id.Data, g.Id.Error
}
