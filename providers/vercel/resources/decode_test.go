// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
)

// The API records in this package are decoded by struct tag alone. A mistyped
// tag compiles, lints, and yields a zero value, which surfaces as a confident
// "false" or "" rather than an error. These tests pin each security-relevant
// tag against a payload shaped like the documented API response.

func TestTeamRecordDecodesGovernanceFields(t *testing.T) {
	const payload = `{
		"id": "team_abc",
		"slug": "acme",
		"name": "Acme",
		"createdAt": 1600000000000,
		"updatedAt": 1600000001000,
		"creatorId": "user_1",
		"billing": {"plan": "enterprise"},
		"parentId": "org_1",
		"orgRootTeamId": "team_root",
		"saml": {
			"enforced": true,
			"roles": {"grp": "OWNER"},
			"connection": {
				"type": "Okta",
				"state": "active",
				"status": "linked",
				"connectedAt": 1600000002000
			},
			"directory": {
				"type": "Okta",
				"state": "active",
				"syncState": "SETUP",
				"connectedAt": 1600000003000,
				"lastSyncedAt": 1600000004000
			}
		},
		"emailDomain": "acme.com",
		"inviteCode": "super-secret-code",
		"dpAccessRequestsMode": "email-domain",
		"requireVerifiedCommits": true,
		"disableRepositoryDispatchEvents": true,
		"strictDeploymentProtectionSettings": {"enabled": true, "updatedAt": 1},
		"strictPasswordProtectionSettings": {"enabled": true, "updatedAt": 1},
		"strictShareableLinks": {"enabled": true, "updatedAt": 1},
		"defaultDeploymentProtection": {
			"passwordProtection": {"deploymentType": "preview"},
			"ssoProtection": {"deploymentType": "all"}
		},
		"defaultRoles": {
			"teamRoles": ["MEMBER"],
			"teamPermissions": ["CreateProject"]
		},
		"deploymentPolicy": {"gitSources": [{"enabled": true, "sources": []}]},
		"defaultExpirationSettings": {"expirationDays": 30, "deploymentsToKeep": 3},
		"sensitiveEnvironmentVariablePolicy": "on",
		"hideIpAddresses": true,
		"hideIpAddressesInLogDrains": true,
		"remoteCaching": {"enabled": true},
		"connect": {"enabled": true},
		"enablePreviewFeedback": "off-force",
		"enableProductionFeedback": "on",
		"previewDeploymentSuffix": "preview.acme.com",
		"stagingPrefix": "staging",
		"personalAccessTokensInvalidatedAt": 1600000005000,
		"appTokensInvalidatedAt": 1600000006000,
		"apiKeysInvalidatedAt": 1600000007000,
		"integrationTokensInvalidatedAt": 1600000008000
	}`

	var rec teamRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if billingPlan(rec.Billing) != "enterprise" {
		t.Errorf("billingPlan = %q, want enterprise", billingPlan(rec.Billing))
	}
	if rec.ParentID != "org_1" || rec.OrgRootTeamID != "team_root" {
		t.Errorf("org hierarchy: parentId=%q orgRootTeamId=%q", rec.ParentID, rec.OrgRootTeamID)
	}

	if rec.Saml == nil || rec.Saml.Connection == nil || rec.Saml.Directory == nil {
		t.Fatal("saml connection and directory must both decode")
	}
	if rec.Saml.Connection.Type != "Okta" || rec.Saml.Connection.State != "active" || rec.Saml.Connection.Status != "linked" {
		t.Errorf("saml connection: %+v", rec.Saml.Connection)
	}
	if rec.Saml.Connection.ConnectedAt.Time() == nil {
		t.Error("saml connectedAt should decode")
	}
	// SETUP means directory events are acknowledged but role mappings are not
	// applied, so it must not be flattened into the same value as ACTIVE.
	if rec.Saml.Directory.SyncState != "SETUP" {
		t.Errorf("directory syncState = %q, want SETUP", rec.Saml.Directory.SyncState)
	}
	if rec.Saml.Directory.LastSynced.Time() == nil {
		t.Error("directory lastSyncedAt should decode")
	}

	if rec.EmailDomain == nil || *rec.EmailDomain != "acme.com" {
		t.Errorf("emailDomain = %v", rec.EmailDomain)
	}
	// The code itself is a credential and must never reach a field; only its
	// presence is reported.
	if rec.InviteCode == "" {
		t.Error("inviteCode should decode so presence can be reported")
	}
	if rec.DpAccessRequestsMode == nil || *rec.DpAccessRequestsMode != "email-domain" {
		t.Errorf("dpAccessRequestsMode = %v", rec.DpAccessRequestsMode)
	}

	for name, got := range map[string]bool{
		"requireVerifiedCommits":             boolPtrOrFalse(rec.RequireVerifiedCommits),
		"disableRepositoryDispatchEvents":    boolPtrOrFalse(rec.DisableRepositoryDispatchEvents),
		"hideIpAddresses":                    boolPtrOrFalse(rec.HideIPAddresses),
		"hideIpAddressesInLogDrains":         boolPtrOrFalse(rec.HideIPAddressesInLogDrains),
		"strictDeploymentProtectionSettings": strictEnabled(rec.StrictDeploymentProtection),
		"strictPasswordProtectionSettings":   strictEnabled(rec.StrictPasswordProtection),
		"strictShareableLinks":               strictEnabled(rec.StrictShareableLinks),
	} {
		if !got {
			t.Errorf("%s decoded false, want true", name)
		}
	}

	if rec.DefaultDeploymentProtection == nil {
		t.Fatal("defaultDeploymentProtection must decode")
	}
	if got := holderType(rec.DefaultDeploymentProtection.SsoProtection); got == nil || *got != "all" {
		t.Errorf("default sso deploymentType = %v", got)
	}
	if got := holderType(rec.DefaultDeploymentProtection.PasswordProtection); got == nil || *got != "preview" {
		t.Errorf("default password deploymentType = %v", got)
	}

	if rec.DefaultRoles == nil || len(rec.DefaultRoles.TeamRoles) != 1 || len(rec.DefaultRoles.TeamPermissions) != 1 {
		t.Errorf("defaultRoles = %+v", rec.DefaultRoles)
	}
	if rec.DeploymentPolicy == nil {
		t.Error("deploymentPolicy should decode")
	}
	if rec.DefaultExpirationSettings == nil || intPtrOrZero(rec.DefaultExpirationSettings.ExpirationDays) != 30 {
		t.Errorf("defaultExpirationSettings = %+v", rec.DefaultExpirationSettings)
	}

	for name, got := range map[string]bool{
		"personalAccessTokensInvalidatedAt": rec.PersonalAccessTokensInvalidatedAt.Time() != nil,
		"appTokensInvalidatedAt":            rec.AppTokensInvalidatedAt.Time() != nil,
		"apiKeysInvalidatedAt":              rec.APIKeysInvalidatedAt.Time() != nil,
		"integrationTokensInvalidatedAt":    rec.IntegrationTokensInvalidatedAt.Time() != nil,
	} {
		if !got {
			t.Errorf("%s did not decode", name)
		}
	}
}

func TestTeamRecordAbsentNestedObjectsStayZero(t *testing.T) {
	// A team on a plan without these features omits the objects entirely. Each
	// extraction is nil-guarded, so this must not panic and must not invent
	// values.
	var rec teamRecord
	if err := json.Unmarshal([]byte(`{"id":"team_abc","slug":"acme"}`), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if billingPlan(rec.Billing) != "" {
		t.Error("absent billing should yield an empty plan")
	}
	if strictEnabled(rec.StrictDeploymentProtection) {
		t.Error("absent strict setting should be false")
	}
	if rec.Saml != nil || rec.DefaultRoles != nil || rec.DefaultExpirationSettings != nil {
		t.Error("absent nested objects should stay nil")
	}
	if boolPtrOrFalse(rec.RequireVerifiedCommits) {
		t.Error("absent requireVerifiedCommits should be false")
	}
}

func TestMemberRecordDecodesRolesAndJoinOrigin(t *testing.T) {
	const payload = `{
		"uid": "user_1",
		"email": "a@acme.com",
		"role": "MEMBER",
		"teamRoles": ["MEMBER", "SECURITY"],
		"teamPermissions": ["EnvVariableManager"],
		"joinedFrom": {"origin": "dsync", "idpUserId": "idp_1"},
		"confirmed": true,
		"accessRequestedAt": 1600000000000,
		"createdAt": 1600000001000
	}`

	var rec memberRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(rec.TeamRoles) != 2 || rec.TeamRoles[1] != "SECURITY" {
		t.Errorf("teamRoles = %v", rec.TeamRoles)
	}
	if len(rec.TeamPermissions) != 1 {
		t.Errorf("teamPermissions = %v", rec.TeamPermissions)
	}
	// Distinguishes an identity-provider provisioned member from one who
	// followed an invite link.
	if rec.JoinedFrom == nil || rec.JoinedFrom.Origin != "dsync" {
		t.Errorf("joinedFrom = %+v", rec.JoinedFrom)
	}
	if rec.AccessRequestedAt.Time() == nil {
		t.Error("accessRequestedAt should decode")
	}
}

func TestSharedEnvRecordDecodes(t *testing.T) {
	// projectId is an array despite the singular key, which is easy to get
	// wrong and would silently drop every project link.
	const payload = `{
		"id": "env_1",
		"key": "DATABASE_URL",
		"type": "sensitive",
		"target": ["production", "preview"],
		"applyToAllCustomEnvironments": true,
		"customEnvironmentIds": ["ce_1"],
		"decrypted": false,
		"comment": "shared db",
		"projectId": ["prj_1", "prj_2"],
		"createdBy": "user_1",
		"updatedBy": "user_2",
		"lastEditedByDisplayName": "Ada",
		"createdAt": 1600000000000,
		"updatedAt": 1600000001000
	}`

	var rec sharedEnvRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(rec.ProjectIDs) != 2 || rec.ProjectIDs[0] != "prj_1" {
		t.Errorf("projectId should decode as a list, got %v", rec.ProjectIDs)
	}
	if rec.Type != "sensitive" {
		t.Errorf("type = %q, want sensitive", rec.Type)
	}
	if len(rec.Target) != 2 || len(rec.CustomEnvironmentIDs) != 1 {
		t.Errorf("target=%v customEnvironmentIds=%v", rec.Target, rec.CustomEnvironmentIDs)
	}
	if !boolPtrOrFalse(rec.ApplyToAllCustomEnvironments) {
		t.Error("applyToAllCustomEnvironments should be true")
	}
	if boolPtrOrFalse(rec.Decrypted) {
		t.Error("decrypted should be false")
	}
}

func TestProjectRecordDecodesSecuritySettings(t *testing.T) {
	const payload = `{
		"id": "prj_1",
		"name": "web",
		"paused": true,
		"tier": "standard",
		"passport": {"deploymentType": "preview", "connectorId": "conn_1"},
		"trustedSources": {"deploymentType": "all"},
		"oidcTokenConfig": {"enabled": true, "issuerMode": "team"},
		"deploymentPolicy": {"deploymentSources": [{"enabled": true, "sources": []}]},
		"rollingRelease": {"target": "canary"},
		"deploymentExpiration": {"expirationDays": 7, "expirationDaysProduction": 90, "deploymentsToKeep": 5},
		"directoryListing": true,
		"sourceFilesOutsideRootDirectory": true,
		"customerSupportCodeVisibility": true,
		"serverlessFunctionZeroConfigFailover": true,
		"autoAssignCustomDomains": true,
		"commandForIgnoringBuildStep": "exit 1",
		"serverlessFunctionRegion": "iad1",
		"resourceConfig": {
			"functionDefaultRegions": ["iad1", "sfo1"],
			"functionDefaultTimeout": 60,
			"functionDefaultMemoryType": "performance",
			"buildMachineType": "enhanced",
			"buildMachineSelection": "elastic"
		},
		"staticIps": {"enabled": true, "builds": true, "regions": ["iad1"]},
		"connectConfigurations": [{
			"connectConfigurationId": "cc_1",
			"envId": "production",
			"dc": "iad1",
			"passive": false,
			"buildsEnabled": true,
			"aws": {"subnetIds": ["subnet-1", "subnet-2"]}
		}],
		"skewProtectionBoundaryAt": 1600000000000,
		"skewProtectionMaxAge": 3600,
		"skewProtectionAllowedDomains": ["acme.com"],
		"transferStartedAt": 1600000001000,
		"transferCompletedAt": 1600000002000,
		"transferToAccountId": "team_b",
		"transferredFromAccountId": "team_a",
		"gitProviderOptions": {
			"createDeployments": "disabled",
			"requireVerifiedCommits": true,
			"disableRepositoryDispatchEvents": true
		},
		"gitComments": {"onCommit": true, "onPullRequest": false}
	}`

	var rec projectRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if rec.Passport == nil || rec.Passport.DeploymentType != "preview" || rec.Passport.ConnectorID != "conn_1" {
		t.Errorf("passport = %+v", rec.Passport)
	}
	if rec.OidcTokenConfig["issuerMode"] != "team" {
		t.Errorf("oidcTokenConfig issuerMode = %v", rec.OidcTokenConfig["issuerMode"])
	}
	if rec.TrustedSources == nil || rec.DeploymentPolicy == nil || rec.RollingRelease == nil {
		t.Error("trustedSources, deploymentPolicy and rollingRelease should all decode")
	}

	if rec.DeploymentExpiration == nil {
		t.Fatal("deploymentExpiration must decode")
	}
	if intPtrOrZero(rec.DeploymentExpiration.ExpirationDaysProduction) != 90 {
		t.Errorf("expirationDaysProduction = %d, want 90", intPtrOrZero(rec.DeploymentExpiration.ExpirationDaysProduction))
	}

	for name, got := range map[string]bool{
		"directoryListing":                     boolPtrOrFalse(rec.DirectoryListing),
		"sourceFilesOutsideRootDirectory":      boolPtrOrFalse(rec.SourceFilesOutsideRootDirectory),
		"customerSupportCodeVisibility":        boolPtrOrFalse(rec.CustomerSupportCodeVisibility),
		"serverlessFunctionZeroConfigFailover": boolPtrOrFalse(rec.ServerlessFunctionZeroConfigFailover),
		"autoAssignCustomDomains":              boolPtrOrFalse(rec.AutoAssignCustomDomains),
		"paused":                               boolPtrOrFalse(rec.Paused),
	} {
		if !got {
			t.Errorf("%s decoded false, want true", name)
		}
	}

	if rec.ResourceConfig == nil || len(rec.ResourceConfig.FunctionDefaultRegions) != 2 {
		t.Fatalf("resourceConfig = %+v", rec.ResourceConfig)
	}
	if intPtrOrZero(rec.ResourceConfig.FunctionDefaultTimeout) != 60 {
		t.Errorf("functionDefaultTimeout = %d", intPtrOrZero(rec.ResourceConfig.FunctionDefaultTimeout))
	}
	if rec.StaticIps == nil || !boolPtrOrFalse(rec.StaticIps.Enabled) || len(rec.StaticIps.Regions) != 1 {
		t.Errorf("staticIps = %+v", rec.StaticIps)
	}

	if len(rec.ConnectConfigurations) != 1 {
		t.Fatalf("connectConfigurations = %v", rec.ConnectConfigurations)
	}
	cc := rec.ConnectConfigurations[0]
	if cc.ConnectConfigurationID != "cc_1" || cc.EnvID != "production" || cc.DC != "iad1" {
		t.Errorf("connect configuration = %+v", cc)
	}
	if cc.AWS == nil || len(cc.AWS.SubnetIDs) != 2 {
		t.Errorf("aws subnetIds = %+v", cc.AWS)
	}

	if intPtrOrZero(rec.SkewProtectionMaxAge) != 3600 || rec.SkewProtectionBoundaryAt.Time() == nil {
		t.Error("skew protection fields should decode")
	}
	if rec.TransferToAccountID != "team_b" || rec.TransferredFromAccountID != "team_a" {
		t.Errorf("transfer accounts: to=%q from=%q", rec.TransferToAccountID, rec.TransferredFromAccountID)
	}

	if rec.GitProviderOptions == nil || !boolPtrOrFalse(rec.GitProviderOptions.RequireVerifiedCommits) {
		t.Errorf("gitProviderOptions = %+v", rec.GitProviderOptions)
	}
	if rec.GitComments == nil || !boolPtrOrFalse(rec.GitComments.OnCommit) || boolPtrOrFalse(rec.GitComments.OnPullRequest) {
		t.Errorf("gitComments = %+v", rec.GitComments)
	}
}

func TestFirewallConfigDecodesRulesetState(t *testing.T) {
	const payload = `{
		"firewallEnabled": true,
		"version": 12,
		"updatedAt": "2026-01-01T00:00:00Z",
		"botIdEnabled": true,
		"logHeaders": ["x-api-key"],
		"managedRules": {"owasp": {"active": true}},
		"crs": {"sqli": {"active": true, "action": "deny"}, "xss": {"active": false, "action": "log"}},
		"rules": [{"id": "r1", "name": "block-bots", "active": true, "action": "deny"}],
		"ips": [{"id": "ip1", "ip": "1.2.3.4", "action": "deny"}]
	}`

	var cfg firewallConfig
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !cfg.FirewallEnabled || intPtrOrZero(cfg.Version) != 12 {
		t.Errorf("firewallEnabled=%v version=%d", cfg.FirewallEnabled, intPtrOrZero(cfg.Version))
	}
	if cfg.UpdatedAt.Time() == nil {
		t.Error("updatedAt should decode from an RFC 3339 string")
	}
	if cfg.BotIDEnabled == nil || !*cfg.BotIDEnabled {
		t.Error("botIdEnabled should decode true")
	}
	if len(cfg.LogHeaders.values) != 1 || cfg.LogHeaders.values[0] != "x-api-key" {
		t.Errorf("logHeaders = %v", cfg.LogHeaders.values)
	}
	crs, ok := cfg.CRS.(map[string]any)
	if !ok || len(crs) != 2 {
		t.Fatalf("crs should decode as an object, got %T", cfg.CRS)
	}
	if len(cfg.Rules) != 1 || len(cfg.IPs) != 1 {
		t.Errorf("rules=%d ips=%d", len(cfg.Rules), len(cfg.IPs))
	}
}

func TestBypassRuleRecordDecodesPascalCaseKeys(t *testing.T) {
	// This endpoint is the only one in the provider that returns PascalCase
	// keys, so the tags here do not follow the pattern used everywhere else.
	const payload = `{
		"Id": "bp_1",
		"OwnerId": "team_1",
		"ProjectId": "prj_1",
		"Ip": "203.0.113.5",
		"Domain": "acme.com",
		"Action": "bypass",
		"IsProjectRule": true,
		"Note": "vendor scanner",
		"ActorId": "user_1",
		"ExpiresAt": 1600000000000,
		"CreatedAt": "2026-01-01T00:00:00Z",
		"UpdatedAt": "2026-01-02T00:00:00Z"
	}`

	var rec bypassRuleRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if rec.ID != "bp_1" || rec.IP != "203.0.113.5" || rec.Domain != "acme.com" {
		t.Errorf("identity fields: %+v", rec)
	}
	// A bypass entry skips every other firewall rule, so mistaking it for a
	// block would invert the finding.
	if rec.Action != "bypass" {
		t.Errorf("action = %q, want bypass", rec.Action)
	}
	if !boolPtrOrFalse(rec.IsProjectRule) {
		t.Error("IsProjectRule should decode true")
	}
	if rec.Note == nil || *rec.Note != "vendor scanner" {
		t.Errorf("note = %v", rec.Note)
	}
	if rec.ActorID != "user_1" {
		t.Errorf("actorId = %q", rec.ActorID)
	}
	if rec.ExpiresAt.Time() == nil || rec.CreatedAt.Time() == nil || rec.UpdatedAt.Time() == nil {
		t.Error("timestamps should decode from both epoch millis and RFC 3339")
	}
}

func TestBypassRuleRecordPermanentEntryHasNoExpiry(t *testing.T) {
	var rec bypassRuleRecord
	if err := json.Unmarshal([]byte(`{"Id":"bp_2","Ip":"1.1.1.1","Action":"bypass","ExpiresAt":null}`), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// A permanent exemption is the higher-risk case, so a null expiry must stay
	// null rather than collapsing to the zero time.
	if rec.ExpiresAt.Time() != nil {
		t.Errorf("null ExpiresAt should stay nil, got %v", rec.ExpiresAt.Time())
	}
}
