// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	admin "cloud.google.com/go/iam/admin/apiv1"
	iamv2 "cloud.google.com/go/iam/apiv2"
	iamv2pb "cloud.google.com/go/iam/apiv2/iampb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	policyanalyzer "google.golang.org/api/policyanalyzer/v1"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
)

func (g *mqlGcpProjectIamService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.iamService", projectId), nil
}

type mqlGcpProjectIamServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) iam() (*mqlGcpProjectIamService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_iam)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectIamService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_iam).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) id() (string, error) {
	if g.UniqueId.Error != nil {
		return "", g.UniqueId.Error
	}
	if g.UniqueId.IsNull() || g.UniqueId.Data == "" {
		// Fall back to email so not-found / asset-context instances still get a
		// stable, non-colliding cache key.
		if g.Email.Error != nil {
			return "", g.Email.Error
		}
		if !g.Email.IsNull() && g.Email.Data != "" {
			return "gcp.project.iamService.serviceAccount/email/" + g.Email.Data, nil
		}
		return "", errors.New("service account has no uniqueId or email for cache key")
	}
	return g.UniqueId.Data, nil
}

func (g *mqlGcpProjectIamServiceServiceAccountKey) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func initGcpProjectIamServiceServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID. This happens
	// when the resource is initialized from a gcp-iam-service-account asset.
	// The platform ID encodes the service account's uniqueId as the name segment
	// (see discovery.go and NewResourcePlatformID).
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["projectId"] = llx.StringData(ids.project)
			args["uniqueId"] = llx.StringData(ids.name)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	if args["projectId"] == nil {
		return args, nil, nil
	}
	if args["email"] == nil && args["uniqueId"] == nil {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "gcp.project.iamService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	iamSvc := obj.(*mqlGcpProjectIamService)
	sas := iamSvc.GetServiceAccounts()
	if sas.Error != nil {
		return nil, nil, sas.Error
	}

	for _, s := range sas.Data {
		sa := s.(*mqlGcpProjectIamServiceServiceAccount)

		if args["email"] != nil {
			email := sa.GetEmail()
			if email.Error != nil {
				return nil, nil, email.Error
			}
			if email.Data == args["email"].Value {
				return args, sa, nil
			}
		}

		if args["uniqueId"] != nil {
			uniqueId := sa.GetUniqueId()
			if uniqueId.Error != nil {
				return nil, nil, uniqueId.Error
			}
			if uniqueId.Data == args["uniqueId"].Value {
				return args, sa, nil
			}
		}
	}

	// Capture the original lookup values before we null out missing fields,
	// so the "not found" log reflects what we actually searched for.
	var lookupEmail, lookupUniqueId any
	if args["email"] != nil {
		lookupEmail = args["email"].Value
	}
	if args["uniqueId"] != nil {
		lookupUniqueId = args["uniqueId"].Value
	}

	// Not found: null out all fields so downstream field access returns null
	// instead of hitting "cannot convert primitive with NO type information".
	if args["name"] == nil {
		args["name"] = llx.NilData
	}
	if args["uniqueId"] == nil {
		args["uniqueId"] = llx.NilData
	}
	if args["email"] == nil {
		args["email"] = llx.NilData
	}
	if args["displayName"] == nil {
		args["displayName"] = llx.NilData
	}
	if args["description"] == nil {
		args["description"] = llx.NilData
	}
	if args["oauth2ClientId"] == nil {
		args["oauth2ClientId"] = llx.NilData
	}
	if args["disabled"] == nil {
		args["disabled"] = llx.NilData
	}
	log.Error().
		Interface("email", lookupEmail).
		Interface("uniqueId", lookupUniqueId).
		Err(errors.New("service account not found")).
		Send()
	return args, nil, nil
}

func (g *mqlGcpProjectIamService) serviceAccounts() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	var serviceAccounts []any
	it := adminSvc.ListServiceAccounts(ctx, &adminpb.ListServiceAccountsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlSA, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
			"projectId":      llx.StringData(s.ProjectId),
			"name":           llx.StringData(s.Name),
			"uniqueId":       llx.StringData(s.UniqueId),
			"email":          llx.StringData(s.Email),
			"displayName":    llx.StringData(s.DisplayName),
			"description":    llx.StringData(s.Description),
			"oauth2ClientId": llx.StringData(s.Oauth2ClientId),
			"disabled":       llx.BoolData(s.Disabled),
		})
		if err != nil {
			return nil, err
		}
		serviceAccounts = append(serviceAccounts, mqlSA)
	}
	return serviceAccounts, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) keys() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Email.Error != nil {
		return nil, g.Email.Error
	}
	email := g.Email.Data

	// if the unique id is null, we were not able to find a record of this service account
	// so skip the keys discovery
	if g.UniqueId.IsNull() {
		g.Keys.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	resp, err := adminSvc.ListServiceAccountKeys(ctx, &adminpb.ListServiceAccountKeysRequest{Name: fmt.Sprintf("projects/%s/serviceAccounts/%s", projectId, email)})
	if err != nil {
		return nil, err
	}
	mqlKeys := make([]any, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		keyType := k.KeyType.String()
		mqlKey, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount.key", map[string]*llx.RawData{
			"name":            llx.StringData(k.Name),
			"keyAlgorithm":    llx.StringData(k.KeyAlgorithm.String()),
			"validAfterTime":  llx.TimeDataPtr(timestampAsTimePtr(k.ValidAfterTime)),
			"validBeforeTime": llx.TimeDataPtr(timestampAsTimePtr(k.ValidBeforeTime)),
			"keyOrigin":       llx.StringData(k.KeyOrigin.String()),
			"keyType":         llx.StringData(keyType),
			"userManaged":     llx.BoolData(keyType == "USER_MANAGED"),
			"disabled":        llx.BoolData(k.Disabled),
		})
		if err != nil {
			return nil, err
		}
		mqlKeys = append(mqlKeys, mqlKey)
	}
	return mqlKeys, nil
}

// lastAuthActivity matches the JSON shape returned by the Policy Analyzer
// serviceAccountLastAuthentication activity type.
type lastAuthActivity struct {
	ServiceAccount struct {
		ProjectNumber    string `json:"projectNumber"`
		FullResourceName string `json:"fullResourceName"`
	} `json:"serviceAccount"`
	LastAuthenticatedTime string `json:"lastAuthenticatedTime"`
}

func (g *mqlGcpProjectIamServiceServiceAccount) hasUserManagedKeys() (bool, error) {
	keys, err := g.activeUserManagedKeys()
	if err != nil {
		return false, err
	}
	return len(keys) > 0, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) activeUserManagedKeys() ([]any, error) {
	keys := g.GetKeys()
	if keys.Error != nil {
		return nil, keys.Error
	}
	res := make([]any, 0)
	for _, raw := range keys.Data {
		key, ok := raw.(*mqlGcpProjectIamServiceServiceAccountKey)
		if !ok || key == nil {
			continue
		}
		userManaged := key.GetUserManaged()
		if userManaged.Error != nil {
			return nil, userManaged.Error
		}
		if !userManaged.Data {
			continue
		}
		disabled := key.GetDisabled()
		if disabled.Error != nil {
			return nil, disabled.Error
		}
		if disabled.Data {
			continue
		}
		res = append(res, key)
	}
	return res, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) lastAuthenticatedTime() (*time.Time, error) {
	// If the SA wasn't found at init time, UniqueId is null; skip the call.
	if g.UniqueId.IsNull() {
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Email.Error != nil {
		return nil, g.Email.Error
	}
	email := g.Email.Data
	if email == "" {
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(policyanalyzer.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	paSvc, err := policyanalyzer.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/global/activityTypes/serviceAccountLastAuthentication", projectId)
	filter := fmt.Sprintf("activities.full_resource_name=\"//iam.googleapis.com/projects/%s/serviceAccounts/%s\"", projectId, email)
	resp, err := paSvc.Projects.Locations.ActivityTypes.Activities.Query(parent).Filter(filter).Context(ctx).Do()
	if err != nil {
		// Gracefully degrade if the Policy Analyzer API is disabled or the
		// caller lacks the policyanalyzer.* permissions.
		if gerr, ok := err.(*googleapi.Error); ok && (gerr.Code == 403 || gerr.Code == 404) {
			log.Debug().Str("email", email).Int("code", gerr.Code).Msg("policyanalyzer lastAuthenticatedTime unavailable")
			g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	var latest *time.Time
	for _, a := range resp.Activities {
		if a == nil || len(a.Activity) == 0 {
			continue
		}
		var parsed lastAuthActivity
		if err := json.Unmarshal(a.Activity, &parsed); err != nil {
			log.Debug().Err(err).Msg("failed to parse lastAuthenticatedTime activity")
			continue
		}
		if parsed.LastAuthenticatedTime == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, parsed.LastAuthenticatedTime)
		if err != nil {
			log.Debug().Err(err).Str("value", parsed.LastAuthenticatedTime).Msg("failed to parse lastAuthenticatedTime timestamp")
			continue
		}
		if latest == nil || t.After(*latest) {
			latest = &t
		}
	}

	if latest == nil {
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return latest, nil
}

func (g *mqlGcpProjectIamServiceDenyPolicy) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectIamService) denyPolicies() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(iamv2.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := iamv2.NewPoliciesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// IAM v2 ListPolicies wants the URL-encoded resource path of the policy's
	// attachment point. For a project that point is the Cloud Resource Manager
	// project resource, whose path separators must be percent-encoded as %2F.
	// The %% in the format string emits a literal %, so for project "my-project"
	// this yields: "policies/cloudresourcemanager.googleapis.com%2Fprojects%2Fmy-project/denypolicies".
	parent := fmt.Sprintf("policies/cloudresourcemanager.googleapis.com%%2Fprojects%%2F%s/denypolicies", projectId)

	var policies []any
	it := client.ListPolicies(ctx, &iamv2pb.ListPoliciesRequest{Parent: parent})
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		rules := make([]any, 0, len(p.Rules))
		for _, r := range p.Rules {
			rule := map[string]any{"description": r.Description}
			if denyRule := r.GetDenyRule(); denyRule != nil {
				rule["deniedPrincipals"] = denyRule.DeniedPrincipals
				rule["exceptionPrincipals"] = denyRule.ExceptionPrincipals
				rule["deniedPermissions"] = denyRule.DeniedPermissions
				rule["exceptionPermissions"] = denyRule.ExceptionPermissions
				if cond := denyRule.DenialCondition; cond != nil {
					rule["denialCondition"] = map[string]any{
						"expression":  cond.Expression,
						"title":       cond.Title,
						"description": cond.Description,
						"location":    cond.Location,
					}
				}
			}
			ruleDict, err := convert.JsonToDict(rule)
			if err != nil {
				return nil, err
			}
			rules = append(rules, ruleDict)
		}

		mqlPolicy, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.denyPolicy", map[string]*llx.RawData{
			"name":        llx.StringData(p.Name),
			"uid":         llx.StringData(p.Uid),
			"displayName": llx.StringData(p.DisplayName),
			"annotations": llx.MapData(convert.MapToInterfaceMap(p.Annotations), types.String),
			"rules":       llx.ArrayData(rules, types.Dict),
			"etag":        llx.StringData(p.Etag),
			"created":     llx.TimeDataPtr(timestampAsTimePtr(p.CreateTime)),
			"updated":     llx.TimeDataPtr(timestampAsTimePtr(p.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		policies = append(policies, mqlPolicy)
	}
	return policies, nil
}
