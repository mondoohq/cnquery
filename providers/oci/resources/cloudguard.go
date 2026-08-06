// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// CloudGuard is a tenancy-level service that only operates in the home region,
// unlike other OCI services that require per-region iteration.
//
// The home region and the configuration are guarded by separate mutexes: the
// configuration fetch needs the home region to build its client, and a single
// shared mutex would deadlock because sync.Mutex is not reentrant.
type mqlOciCloudGuardInternal struct {
	configLock     sync.Mutex
	configFetched  atomic.Bool
	config         *cloudguard.Configuration
	homeRegionLock sync.Mutex
	homeRegionSet  atomic.Bool
	homeRegion     string
}

func (o *mqlOciCloudGuard) id() (string, error) {
	return "oci.cloudGuard", nil
}

func (o *mqlOciCloudGuard) getHomeRegion() (string, error) {
	if o.homeRegionSet.Load() {
		return o.homeRegion, nil
	}
	o.homeRegionLock.Lock()
	defer o.homeRegionLock.Unlock()
	if o.homeRegionSet.Load() {
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
	o.homeRegionSet.Store(true)
	return o.homeRegion, nil
}

func (o *mqlOciCloudGuard) getConfig() (*cloudguard.Configuration, error) {
	if o.configFetched.Load() {
		return o.config, nil
	}

	// Resolve the home region before taking configLock: getHomeRegion takes its
	// own lock, and nesting it inside this critical section would deadlock.
	homeRegion, err := o.getHomeRegion()
	if err != nil {
		return nil, err
	}

	o.configLock.Lock()
	defer o.configLock.Unlock()
	if o.configFetched.Load() {
		return o.config, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

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
	o.configFetched.Store(true)
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
			// Cloud Guard targets are attached to sub-compartments far more
			// often than to the tenancy root.
			CompartmentIdInSubtree: common.Bool(true),
			Page:                   page,
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
			"systemTags":         llx.MapData(definedTagsToAny(target.SystemTags), types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

// problems lists every finding Cloud Guard has raised. The listing is
// deliberately unfiltered: `lifecycleDetail` is exposed as a field so callers
// decide whether they care about OPEN findings only or also want the
// RESOLVED and DISMISSED history. Filtering to OPEN here would make a
// dismissed-but-unfixed finding indistinguishable from one that never existed.
func (o *mqlOciCloudGuard) problems() ([]any, error) {
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
	problems := []cloudguard.ProblemSummary{}
	var page *string
	for {
		response, err := client.ListProblems(ctx, cloudguard.ListProblemsRequest{
			CompartmentId: common.String(conn.TenantID()),
			// Problems are raised against resources in sub-compartments, so
			// without the subtree flag the tenancy root reports almost none.
			CompartmentIdInSubtree: common.Bool(true),
			// ACCESSIBLE degrades to the compartments the caller can read
			// instead of failing the whole listing on the first one it cannot.
			AccessLevel: cloudguard.ListProblemsAccessLevelAccessible,
			Page:        page,
		})
		if err != nil {
			return nil, err
		}

		problems = append(problems, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(problems))
	for i := range problems {
		problem := problems[i]

		var riskScore float64
		if problem.RiskScore != nil {
			riskScore = *problem.RiskScore
		}

		mqlProblem, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.problem", map[string]*llx.RawData{
			"id":              llx.StringDataPtr(problem.Id),
			"riskLevel":       llx.StringData(string(problem.RiskLevel)),
			"riskScore":       llx.FloatData(riskScore),
			"detectorRuleId":  llx.StringDataPtr(problem.DetectorRuleId),
			"detector":        llx.StringData(string(problem.DetectorId)),
			"resourceId":      llx.StringDataPtr(problem.ResourceId),
			"resourceName":    llx.StringDataPtr(problem.ResourceName),
			"resourceType":    llx.StringDataPtr(problem.ResourceType),
			"labels":          llx.ArrayData(stringsToAny(problem.Labels), types.String),
			"state":           llx.StringData(string(problem.LifecycleState)),
			"lifecycleDetail": llx.StringData(string(problem.LifecycleDetail)),
			"region":          llx.StringDataPtr(problem.Region),
			"regions":         llx.ArrayData(stringsToAny(problem.Regions), types.String),
			"firstDetected":   sdkTimeData(problem.TimeFirstDetected),
			"lastDetected":    sdkTimeData(problem.TimeLastDetected),
		})
		if err != nil {
			return nil, err
		}
		mqlProblemTyped := mqlProblem.(*mqlOciCloudGuardProblem)
		mqlProblemTyped.cacheCompartmentId = stringValue(problem.CompartmentId)
		mqlProblemTyped.cacheTargetId = stringValue(problem.TargetId)
		res = append(res, mqlProblemTyped)
	}

	return res, nil
}

type mqlOciCloudGuardProblemInternal struct {
	cacheCompartmentId string
	cacheTargetId      string
}

func (o *mqlOciCloudGuardProblem) id() (string, error) {
	return "oci.cloudGuard.problem/" + o.Id.Data, nil
}

func (o *mqlOciCloudGuardProblem) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciCloudGuardProblem) target() (*mqlOciCloudGuardTarget, error) {
	if o.cacheTargetId == "" {
		o.Target.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	// Targets are not individually addressable by OCID through an init, so
	// match against the already-cached target listing rather than issuing a
	// per-problem lookup.
	obj, err := CreateResource(o.MqlRuntime, "oci.cloudGuard", nil)
	if err != nil {
		return nil, err
	}
	rawTargets := obj.(*mqlOciCloudGuard).GetTargets()
	if rawTargets.Error != nil {
		return nil, rawTargets.Error
	}

	for _, raw := range rawTargets.Data {
		t, ok := raw.(*mqlOciCloudGuardTarget)
		if ok && t.Id.Data == o.cacheTargetId {
			return t, nil
		}
	}

	o.Target.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (o *mqlOciCloudGuardDetectorRecipe) rules() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	obj, err := CreateResource(o.MqlRuntime, "oci.cloudGuard", nil)
	if err != nil {
		return nil, err
	}
	homeRegion, err := obj.(*mqlOciCloudGuard).getHomeRegion()
	if err != nil {
		return nil, err
	}

	client, err := conn.CloudGuardClient(homeRegion)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	rules := []cloudguard.DetectorRecipeDetectorRuleSummary{}
	var page *string
	for {
		response, err := client.ListDetectorRecipeDetectorRules(ctx, cloudguard.ListDetectorRecipeDetectorRulesRequest{
			DetectorRecipeId: common.String(o.Id.Data),
			CompartmentId:    common.String(o.CompartmentID.Data),
			Page:             page,
		})
		if err != nil {
			return nil, err
		}

		rules = append(rules, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(rules))
	for i := range rules {
		rule := rules[i]

		// Every field that decides whether the rule actually fires lives in
		// DetectorDetails. A nil block means the API returned the rule without
		// its configuration, so report it as disabled-unknown rather than
		// defaulting isEnabled to false, which would read as an authoritative
		// "this detection is off".
		var (
			isEnabled              bool
			riskLevel              string
			problemThreshold       int64
			labels                 []string
			isConfigurationAllowed bool
			configurations         []any
		)
		if rule.DetectorDetails != nil {
			isEnabled = boolValue(rule.DetectorDetails.IsEnabled)
			riskLevel = string(rule.DetectorDetails.RiskLevel)
			if rule.DetectorDetails.ProblemThreshold != nil {
				problemThreshold = int64(*rule.DetectorDetails.ProblemThreshold)
			}
			labels = rule.DetectorDetails.Labels
			isConfigurationAllowed = boolValue(rule.DetectorDetails.IsConfigurationAllowed)

			configurations, err = convert.JsonToDictSlice(rule.DetectorDetails.Configurations)
			if err != nil {
				return nil, err
			}
		}

		managedListTypes := make([]string, 0, len(rule.ManagedListTypes))
		for _, t := range rule.ManagedListTypes {
			managedListTypes = append(managedListTypes, string(t))
		}

		mqlRule, err := CreateResource(o.MqlRuntime, "oci.cloudGuard.detectorRule", map[string]*llx.RawData{
			"__id":                   llx.StringData(o.Id.Data + "/" + stringValue(rule.Id)),
			"id":                     llx.StringDataPtr(rule.Id),
			"detector":               llx.StringData(string(rule.Detector)),
			"name":                   llx.StringDataPtr(rule.DisplayName),
			"description":            llx.StringDataPtr(rule.Description),
			"recommendation":         llx.StringDataPtr(rule.Recommendation),
			"serviceType":            llx.StringDataPtr(rule.ServiceType),
			"resourceType":           llx.StringDataPtr(rule.ResourceType),
			"isEnabled":              llx.BoolData(isEnabled),
			"riskLevel":              llx.StringData(riskLevel),
			"problemThreshold":       llx.IntData(problemThreshold),
			"labels":                 llx.ArrayData(stringsToAny(labels), types.String),
			"configurations":         llx.ArrayData(configurations, types.Dict),
			"isConfigurationAllowed": llx.BoolData(isConfigurationAllowed),
			"isCloneable":            llx.BoolData(boolValue(rule.IsCloneable)),
			"managedListTypes":       llx.ArrayData(stringsToAny(managedListTypes), types.String),
			"state":                  llx.StringData(string(rule.LifecycleState)),
			"created":                sdkTimeData(rule.TimeCreated),
			"updated":                sdkTimeData(rule.TimeUpdated),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
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
			CompartmentId:          common.String(conn.TenantID()),
			CompartmentIdInSubtree: common.Bool(true),
			Page:                   page,
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
			"id":            llx.StringDataPtr(recipe.Id),
			"name":          llx.StringDataPtr(recipe.DisplayName),
			"description":   llx.StringDataPtr(recipe.Description),
			"compartmentID": llx.StringDataPtr(recipe.CompartmentId),
			"owner":         llx.StringData(string(recipe.Owner)),
			"detectorType":  llx.StringData(string(recipe.Detector)),
			"state":         llx.StringData(string(recipe.LifecycleState)),
			"created":       llx.TimeDataPtr(created),
			"systemTags":    llx.MapData(definedTagsToAny(recipe.SystemTags), types.Dict),
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
		// Security zones are attached to compartments and practically never to
		// the tenancy root, so without the subtree flag a correctly zoned
		// tenancy reported none at all.
		response, err := client.ListSecurityZones(ctx, cloudguard.ListSecurityZonesRequest{
			CompartmentId:                    common.String(conn.TenantID()),
			IsRequiredSecurityZonesInSubtree: common.Bool(true),
			Page:                             page,
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
			"systemTags":                      llx.MapData(definedTagsToAny(zone.SystemTags), types.Dict),
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
			"systemTags":    llx.MapData(definedTagsToAny(recipe.SystemTags), types.Dict),
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
			"systemTags":    llx.MapData(definedTagsToAny(policy.SystemTags), types.Dict),
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
