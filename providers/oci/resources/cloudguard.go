// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// CloudGuard is a tenancy-level service that only operates in the home region,
// unlike other OCI services that require per-region iteration.
type mqlOciCloudGuardInternal struct {
	lock       sync.Mutex
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
	o.lock.Lock()
	defer o.lock.Unlock()
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

	// HomeRegionKey returns the short region key (e.g., "IAD"), not the region name (e.g., "us-ashburn-1").
	// The OCI SDK's SetRegion() accepts both formats.
	o.homeRegion = *tenancy.HomeRegionKey
	return o.homeRegion, nil
}

func (o *mqlOciCloudGuard) getConfig() (*cloudguard.Configuration, error) {
	if o.config != nil {
		return o.config, nil
	}
	o.lock.Lock()
	defer o.lock.Unlock()
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

func (o *mqlOciCloudGuard) securityZones() ([]any, error) {
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
	zones := []cloudguard.SecurityZoneSummary{}
	var page *string
	for {
		response, err := client.ListSecurityZones(ctx, cloudguard.ListSecurityZonesRequest{
			CompartmentId: common.String(conn.TenantID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		zones = append(zones, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(zones))
	for i := range zones {
		zone := zones[i]

		var created *time.Time
		if zone.TimeCreated != nil {
			created = &zone.TimeCreated.Time
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.securityZone", map[string]*llx.RawData{
			"id":                              llx.StringDataPtr(zone.Id),
			"name":                            llx.StringDataPtr(zone.DisplayName),
			"description":                     llx.StringDataPtr(zone.Description),
			"compartmentID":                   llx.StringDataPtr(zone.CompartmentId),
			"isInheritanceAfterDeleteEnabled": llx.BoolDataPtr(zone.IsInheritanceAfterDeleteEnabled),
			"state":                           llx.StringData(string(zone.LifecycleState)),
			"created":                         llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		mqlZone := mqlInstance.(*mqlOciCloudGuardSecurityZone)
		mqlZone.cacheRecipeId = stringValue(zone.SecurityZoneRecipeId)
		res = append(res, mqlZone)
	}

	return res, nil
}

func (o *mqlOciCloudGuard) securityZoneRecipes() ([]any, error) {
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
	recipes := []cloudguard.SecurityRecipeSummary{}
	var page *string
	for {
		response, err := client.ListSecurityRecipes(ctx, cloudguard.ListSecurityRecipesRequest{
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

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.securityZoneRecipe", map[string]*llx.RawData{
			"id":            llx.StringDataPtr(recipe.Id),
			"name":          llx.StringDataPtr(recipe.DisplayName),
			"description":   llx.StringDataPtr(recipe.Description),
			"compartmentID": llx.StringDataPtr(recipe.CompartmentId),
			"owner":         llx.StringData(string(recipe.Owner)),
			"state":         llx.StringData(string(recipe.LifecycleState)),
			"created":       llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		mqlRecipe := mqlInstance.(*mqlOciCloudGuardSecurityZoneRecipe)
		mqlRecipe.cachePolicyIds = recipe.SecurityPolicies
		res = append(res, mqlRecipe)
	}

	return res, nil
}

func (o *mqlOciCloudGuard) securityPolicies() ([]any, error) {
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
	policies := []cloudguard.SecurityPolicySummary{}
	var page *string
	for {
		response, err := client.ListSecurityPolicies(ctx, cloudguard.ListSecurityPoliciesRequest{
			CompartmentId: common.String(conn.TenantID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}

		policies = append(policies, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(policies))
	for i := range policies {
		policy := policies[i]

		var created *time.Time
		if policy.TimeCreated != nil {
			created = &policy.TimeCreated.Time
		}

		services := make([]any, 0, len(policy.Services))
		for _, s := range policy.Services {
			services = append(services, s)
		}

		mqlInstance, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.securityPolicy", map[string]*llx.RawData{
			"id":            llx.StringDataPtr(policy.Id),
			"name":          llx.StringDataPtr(policy.DisplayName),
			"friendlyName":  llx.StringDataPtr(policy.FriendlyName),
			"description":   llx.StringDataPtr(policy.Description),
			"compartmentID": llx.StringDataPtr(policy.CompartmentId),
			"owner":         llx.StringData(string(policy.Owner)),
			"category":      llx.StringDataPtr(policy.Category),
			"services":      llx.ArrayData(services, types.String),
			"state":         llx.StringData(string(policy.LifecycleState)),
			"created":       llx.TimeDataPtr(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

type mqlOciCloudGuardSecurityZoneInternal struct {
	cacheRecipeId string
}

type mqlOciCloudGuardSecurityZoneRecipeInternal struct {
	cachePolicyIds []string
}

func (o *mqlOciCloudGuardTarget) id() (string, error) {
	return "oci.cloudGuard.target/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardDetectorRecipe) id() (string, error) {
	return "oci.cloudGuard.detectorRecipe/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardSecurityZone) id() (string, error) {
	return "oci.cloudGuard.securityZone/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardSecurityZone) securityZoneRecipe() (*mqlOciCloudGuardSecurityZoneRecipe, error) {
	if o.cacheRecipeId == "" {
		o.SecurityZoneRecipe.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	cgRes, err := CreateResource(o.MqlRuntime, "oci.cloudGuard", nil)
	if err != nil {
		return nil, err
	}
	cg := cgRes.(*mqlOciCloudGuard)

	rawRecipes := cg.GetSecurityZoneRecipes()
	if rawRecipes.Error != nil {
		return nil, rawRecipes.Error
	}

	for _, raw := range rawRecipes.Data {
		r := raw.(*mqlOciCloudGuardSecurityZoneRecipe)
		if r.Id.Data == o.cacheRecipeId {
			return r, nil
		}
	}

	o.SecurityZoneRecipe.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (o *mqlOciCloudGuardSecurityZoneRecipe) id() (string, error) {
	return "oci.cloudGuard.securityZoneRecipe/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardSecurityZoneRecipe) securityPolicies() ([]any, error) {
	if len(o.cachePolicyIds) == 0 {
		return []any{}, nil
	}

	cgRes, err := CreateResource(o.MqlRuntime, "oci.cloudGuard", nil)
	if err != nil {
		return nil, err
	}
	cg := cgRes.(*mqlOciCloudGuard)

	rawPolicies := cg.GetSecurityPolicies()
	if rawPolicies.Error != nil {
		return nil, rawPolicies.Error
	}

	byId := make(map[string]*mqlOciCloudGuardSecurityPolicy, len(rawPolicies.Data))
	for _, raw := range rawPolicies.Data {
		p := raw.(*mqlOciCloudGuardSecurityPolicy)
		byId[p.Id.Data] = p
	}

	res := make([]any, 0, len(o.cachePolicyIds))
	for _, id := range o.cachePolicyIds {
		if p, ok := byId[id]; ok {
			res = append(res, p)
		}
	}
	return res, nil
}

func (o *mqlOciCloudGuardSecurityPolicy) id() (string, error) {
	return "oci.cloudGuard.securityPolicy/" + o.Id.Data, nil
}
