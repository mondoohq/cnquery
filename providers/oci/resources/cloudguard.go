// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/oci/connection"
)

type mqlOciCloudGuardInternal struct {
	config     *cloudguard.Configuration
	homeRegion string
}

func (o *mqlOciCloudGuard) id() (string, error) {
	return "oci.cloudGuard", nil
}

func (o *mqlOciCloudGuard) getHomeRegion() (string, error) {
	if o.homeRegion != "" {
		return o.homeRegion, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	tenancy, err := conn.Tenant(context.Background())
	if err != nil {
		return "", err
	}

	if tenancy.HomeRegionKey == nil {
		return "", errors.New("no home region set")
	}

	o.homeRegion = *tenancy.HomeRegionKey
	return o.homeRegion, nil
}

func (o *mqlOciCloudGuard) getConfig() (*cloudguard.Configuration, error) {
	if o.config != nil {
		return o.config, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	homeRegion, err := o.getHomeRegion()
	if err != nil {
		return nil, err
	}

	client, err := conn.CloudGuardClient(homeRegion)
	if err != nil {
		return nil, err
	}

	response, err := client.GetConfiguration(context.Background(), cloudguard.GetConfigurationRequest{
		CompartmentId: common.String(conn.TenantID()),
	})
	if err != nil {
		return nil, err
	}

	o.config = &response.Configuration
	return o.config, nil
}

func (o *mqlOciCloudGuard) status() (bool, error) {
	cfg, err := o.getConfig()
	if err != nil {
		return false, err
	}
	return cfg.Status == cloudguard.CloudGuardStatusEnabled, nil
}

func (o *mqlOciCloudGuard) reportingRegion() (string, error) {
	cfg, err := o.getConfig()
	if err != nil {
		return "", err
	}
	return stringValue(cfg.ReportingRegion), nil
}

func (o *mqlOciCloudGuard) selfManageResources() (bool, error) {
	cfg, err := o.getConfig()
	if err != nil {
		return false, err
	}
	return boolValue(cfg.SelfManageResources), nil
}

func (o *mqlOciCloudGuard) targets() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	homeRegion, err := o.getHomeRegion()
	if err != nil {
		return nil, err
	}

	client, err := conn.CloudGuardClient(homeRegion)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	targets := []cloudguard.TargetSummary{}
	var page *string
	for {
		response, err := client.ListTargets(ctx, cloudguard.ListTargetsRequest{
			CompartmentId: common.String(conn.TenantID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		targets = append(targets, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(targets))
	for i := range targets {
		target := targets[i]

		var created *time.Time
		if target.TimeCreated != nil {
			created = &target.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.target", map[string]*llx.RawData{
			"id":                 llx.StringDataPtr(target.Id),
			"name":               llx.StringDataPtr(target.DisplayName),
			"compartmentID":      llx.StringDataPtr(target.CompartmentId),
			"targetResourceId":   llx.StringDataPtr(target.TargetResourceId),
			"targetResourceType": llx.StringData(string(target.TargetResourceType)),
			"state":              llx.StringData(string(target.LifecycleState)),
			"recipeCount":        llx.IntDataPtr(target.RecipeCount),
			"created":            llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciCloudGuard) detectorRecipes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	homeRegion, err := o.getHomeRegion()
	if err != nil {
		return nil, err
	}

	client, err := conn.CloudGuardClient(homeRegion)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	recipes := []cloudguard.DetectorRecipeSummary{}
	var page *string
	for {
		response, err := client.ListDetectorRecipes(ctx, cloudguard.ListDetectorRecipesRequest{
			CompartmentId: common.String(conn.TenantID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		recipes = append(recipes, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(recipes))
	for i := range recipes {
		recipe := recipes[i]

		var created *time.Time
		if recipe.TimeCreated != nil {
			created = &recipe.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.detectorRecipe", map[string]*llx.RawData{
			"id":           llx.StringDataPtr(recipe.Id),
			"name":         llx.StringDataPtr(recipe.DisplayName),
			"description":  llx.StringDataPtr(recipe.Description),
			"owner":        llx.StringData(string(recipe.Owner)),
			"detectorType": llx.StringData(string(recipe.Detector)),
			"state":        llx.StringData(string(recipe.LifecycleState)),
			"created":      llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (o *mqlOciCloudGuardTarget) id() (string, error) {
	return "oci.cloudGuard.target/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardDetectorRecipe) id() (string, error) {
	return "oci.cloudGuard.detectorRecipe/" + o.Id.Data, nil
}
