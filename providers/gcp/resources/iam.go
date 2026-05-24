// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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

type mqlGcpProjectIamServiceServiceAccountInternal struct {
	keyLastAuthOnce sync.Once
	keyLastAuthMap  map[string]*time.Time
	keyLastAuthErr  error
}

type mqlGcpProjectIamServiceServiceAccountKeyInternal struct {
	// Back-reference set when the key is loaded via the parent SA's keys()
	// accessor. Nil when the key is constructed in isolation (e.g., from a
	// platform-ID asset import), in which case lastAuthenticatedTime() falls
	// back to a per-key Policy Analyzer query.
	parentSA *mqlGcpProjectIamServiceServiceAccount
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

// Direct construction (e.g. `gcp.project.iamService.serviceAccounts`) bypasses
// gcp.project.iam(), so projectId stays empty and serviceEnabled stays false —
// accessors then short-circuit and silently return empty. Delegate to the
// parent project accessor so the resulting instance is fully initialized.
func initGcpProjectIamService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["projectId"]; ok {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	proj, err := NewResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id": llx.StringData(conn.ResourceID()),
	})
	if err != nil {
		return nil, nil, err
	}
	svc, err := proj.(*mqlGcpProject).iam()
	if err != nil {
		return nil, nil, err
	}
	return nil, svc, nil
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

	// Go through gcp.project so the iamService picks up its `serviceEnabled`
	// flag — constructing the iamService directly leaves the flag at its
	// zero value, which short-circuits serviceAccounts() to nil and makes the
	// SA lookup miss every typed-ref query.
	projObj, err := CreateResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	iamRes := projObj.(*mqlGcpProject).GetIam()
	if iamRes.Error != nil {
		return nil, nil, iamRes.Error
	}
	iamSvc := iamRes.Data
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
		// Back-reference the parent SA so key.lastAuthenticatedTime() can
		// reuse a single batched Policy Analyzer query per SA instead of
		// querying once per key.
		mqlKey.(*mqlGcpProjectIamServiceServiceAccountKey).parentSA = g
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

// lastAuthKeyActivity matches the JSON shape returned by the Policy Analyzer
// serviceAccountKeyLastAuthentication activity type.
type lastAuthKeyActivity struct {
	ServiceAccountKey struct {
		ProjectNumber    string `json:"projectNumber"`
		FullResourceName string `json:"fullResourceName"`
	} `json:"serviceAccountKey"`
	LastAuthenticatedTime string `json:"lastAuthenticatedTime"`
}

// fetchKeyLastAuthTimes batch-queries Policy Analyzer for the last-auth
// timestamp of every key on this service account in a single Activities.Query
// call. Returns a map keyed by full key resource name
// (projects/{p}/serviceAccounts/{email}/keys/{keyId}) to the latest auth time
// observed for that key. Subsequent calls return the cached result.
func (g *mqlGcpProjectIamServiceServiceAccount) fetchKeyLastAuthTimes() (map[string]*time.Time, error) {
	g.keyLastAuthOnce.Do(func() {
		g.keyLastAuthMap = map[string]*time.Time{}

		if g.ProjectId.Error != nil {
			g.keyLastAuthErr = g.ProjectId.Error
			return
		}
		if g.Email.Error != nil {
			g.keyLastAuthErr = g.Email.Error
			return
		}
		projectId := g.ProjectId.Data
		email := g.Email.Data
		if email == "" {
			return
		}

		conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
		client, err := conn.Client(policyanalyzer.CloudPlatformScope)
		if err != nil {
			g.keyLastAuthErr = err
			return
		}

		ctx := context.Background()
		paSvc, err := policyanalyzer.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			g.keyLastAuthErr = err
			return
		}

		// The Policy Analyzer key-last-auth filter only supports equality on a
		// single key's full_resource_name, not a service-account-prefix match.
		// To batch the lookup we fetch every key-last-auth activity in the
		// project and filter to this SA's keys client-side; the SA-level
		// sync.Once ensures we only pay for one API call per SA even when many
		// keys' lastAuthenticatedTime is requested.
		parent := fmt.Sprintf("projects/%s/locations/global/activityTypes/serviceAccountKeyLastAuthentication", projectId)
		saPrefix := fmt.Sprintf("//iam.googleapis.com/projects/%s/serviceAccounts/%s/keys/", projectId, email)

		call := paSvc.Projects.Locations.ActivityTypes.Activities.Query(parent).Context(ctx)
		err = call.Pages(ctx, func(resp *policyanalyzer.GoogleCloudPolicyanalyzerV1QueryActivityResponse) error {
			for _, a := range resp.Activities {
				if a == nil || len(a.Activity) == 0 {
					continue
				}
				var parsed lastAuthKeyActivity
				if err := json.Unmarshal(a.Activity, &parsed); err != nil {
					log.Debug().Err(err).Msg("failed to parse key lastAuthenticatedTime activity")
					continue
				}
				if !strings.HasPrefix(parsed.ServiceAccountKey.FullResourceName, saPrefix) {
					continue
				}
				if parsed.LastAuthenticatedTime == "" {
					continue
				}
				t, err := time.Parse(time.RFC3339, parsed.LastAuthenticatedTime)
				if err != nil {
					log.Debug().Err(err).Str("value", parsed.LastAuthenticatedTime).Msg("failed to parse key lastAuthenticatedTime timestamp")
					continue
				}
				// The FRN is "//iam.googleapis.com/{name}" — strip the prefix to
				// match the key resource's `name` field.
				keyName := strings.TrimPrefix(parsed.ServiceAccountKey.FullResourceName, "//iam.googleapis.com/")
				if existing, ok := g.keyLastAuthMap[keyName]; !ok || t.After(*existing) {
					tc := t
					g.keyLastAuthMap[keyName] = &tc
				}
			}
			return nil
		})
		if err != nil {
			if gerr, ok := err.(*googleapi.Error); ok && (gerr.Code == 403 || gerr.Code == 404) {
				log.Debug().Str("email", email).Int("code", gerr.Code).Msg("policyanalyzer key lastAuthenticatedTime unavailable")
				return
			}
			g.keyLastAuthErr = err
		}
	})
	return g.keyLastAuthMap, g.keyLastAuthErr
}

// keyLastAuthFallback queries Policy Analyzer for a single key when the
// batched path through the parent SA is unavailable (typically because the
// key was constructed from a platform-ID asset import and has no back-ref).
func (g *mqlGcpProjectIamServiceServiceAccountKey) keyLastAuthFallback(projectId, keyName string) (*time.Time, error) {
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

	parent := fmt.Sprintf("projects/%s/locations/global/activityTypes/serviceAccountKeyLastAuthentication", projectId)
	filter := fmt.Sprintf("activities.full_resource_name=\"//iam.googleapis.com/%s\"", keyName)
	resp, err := paSvc.Projects.Locations.ActivityTypes.Activities.Query(parent).Filter(filter).Context(ctx).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && (gerr.Code == 403 || gerr.Code == 404) {
			log.Debug().Str("key", keyName).Int("code", gerr.Code).Msg("policyanalyzer key lastAuthenticatedTime unavailable")
			return nil, nil
		}
		return nil, err
	}

	var latest *time.Time
	for _, a := range resp.Activities {
		if a == nil || len(a.Activity) == 0 {
			continue
		}
		var parsed lastAuthKeyActivity
		if err := json.Unmarshal(a.Activity, &parsed); err != nil {
			log.Debug().Err(err).Msg("failed to parse key lastAuthenticatedTime activity")
			continue
		}
		if parsed.LastAuthenticatedTime == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, parsed.LastAuthenticatedTime)
		if err != nil {
			log.Debug().Err(err).Str("value", parsed.LastAuthenticatedTime).Msg("failed to parse key lastAuthenticatedTime timestamp")
			continue
		}
		if latest == nil || t.After(*latest) {
			latest = &t
		}
	}
	return latest, nil
}

func (g *mqlGcpProjectIamServiceServiceAccountKey) lastAuthenticatedTime() (*time.Time, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	keyName := g.Name.Data
	if keyName == "" {
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	// Parse projects/{project}/serviceAccounts/{email}/keys/{keyId}.
	parts := strings.Split(keyName, "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "serviceAccounts" || parts[4] != "keys" {
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	projectId := parts[1]

	// Preferred path: reuse the SA's batched Policy Analyzer query.
	if g.parentSA != nil {
		times, err := g.parentSA.fetchKeyLastAuthTimes()
		if err != nil {
			return nil, err
		}
		if t, ok := times[keyName]; ok {
			return t, nil
		}
		g.LastAuthenticatedTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	// Fallback for keys constructed without a parent SA back-reference (e.g.,
	// from a platform-ID asset import).
	latest, err := g.keyLastAuthFallback(projectId, keyName)
	if err != nil {
		return nil, err
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
