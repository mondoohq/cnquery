// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"
)

// The API records in this package are decoded by struct tag alone. A mistyped
// tag compiles, lints, and yields a zero value, which surfaces as a confident
// "false" or "" rather than an error. A permission target whose actions decode
// to nothing reports an instance as safe that is not, so each tag is pinned
// against a payload shaped like the documented response.

func TestPermissionTargetRecordDecodesGrants(t *testing.T) {
	const payload = `{
		"name": "example-publishers",
		"repo": {
			"include-patterns": ["example/**"],
			"exclude-patterns": ["example/private/**"],
			"repositories": ["example-docker", "ANY REMOTE"],
			"actions": {
				"users": {"anonymous": ["read"], "build-account": ["read", "write"]},
				"groups": {"platform-admins": ["read", "write", "manage"]}
			}
		},
		"build": {
			"include-patterns": [""],
			"exclude-patterns": [],
			"repositories": ["artifactory-build-info"],
			"actions": {"groups": {"ci": ["read"]}}
		},
		"releaseBundle": {
			"include-patterns": ["**"],
			"exclude-patterns": [],
			"repositories": ["release-bundles"],
			"actions": {"users": {"release-manager": ["read", "distribute"]}}
		}
	}`

	var rec permissionTargetRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if rec.Name != "example-publishers" {
		t.Errorf("name = %q", rec.Name)
	}
	if rec.Repo == nil || rec.Build == nil || rec.ReleaseBundle == nil {
		t.Fatalf("a scope decoded as absent: %+v", rec)
	}

	repo := rec.Repo
	if len(repo.IncludePatterns) != 1 || repo.IncludePatterns[0] != "example/**" {
		t.Errorf("include patterns decoded wrong: %v", repo.IncludePatterns)
	}
	if len(repo.ExcludePatterns) != 1 || repo.ExcludePatterns[0] != "example/private/**" {
		t.Errorf("exclude patterns decoded wrong: %v", repo.ExcludePatterns)
	}
	if len(repo.Repositories) != 2 || repo.Repositories[1] != "ANY REMOTE" {
		t.Errorf("repositories decoded wrong: %v", repo.Repositories)
	}
	if got := repo.Actions.Users["anonymous"]; len(got) != 1 || got[0] != "read" {
		t.Errorf("anonymous actions decoded wrong: %v", got)
	}
	if got := repo.Actions.Groups["platform-admins"]; len(got) != 3 {
		t.Errorf("group actions decoded wrong: %v", got)
	}
	if got := rec.ReleaseBundle.Actions.Users["release-manager"]; len(got) != 2 {
		t.Errorf("release bundle actions decoded wrong: %v", got)
	}
}

// A permission target that grants over repositories only must leave the other
// scopes absent rather than empty, so a query can tell "no grant" from "a
// grant that names nothing".
func TestPermissionTargetRecordKeepsAbsentScopesNull(t *testing.T) {
	const payload = `{"name":"example","repo":{"repositories":["example-docker"],"actions":{"users":{}}}}`

	var rec permissionTargetRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Repo == nil {
		t.Fatal("repo scope decoded as absent")
	}
	if rec.Build != nil || rec.ReleaseBundle != nil {
		t.Errorf("an absent scope decoded as present: %+v", rec)
	}
}

func TestRepositoryRecordsDecode(t *testing.T) {
	const listPayload = `[
		{"key":"example-docker","description":"images","type":"LOCAL","packageType":"Docker"},
		{"key":"example-remote","type":"REMOTE","url":"https://registry.example.com","packageType":"Docker"}
	]`

	var list []repositoryRecord
	if err := json.Unmarshal([]byte(listPayload), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(list))
	}
	if list[1].URL != "https://registry.example.com" {
		t.Errorf("upstream URL decoded wrong: %q", list[1].URL)
	}

	const detailPayload = `{
		"key": "example-remote",
		"rclass": "remote",
		"packageType": "docker",
		"url": "https://registry.example.com",
		"repositories": ["example-docker", "example-remote"],
		"xrayIndex": true,
		"blackedOut": false,
		"offline": false,
		"includesPattern": "**/*",
		"excludesPattern": "secret/**",
		"repoLayoutRef": "simple-default",
		"allowAnyHostAuth": true,
		"externalDependenciesEnabled": true
	}`

	var detail repositoryDetailRecord
	if err := json.Unmarshal([]byte(detailPayload), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.RClass != "remote" {
		t.Errorf("rclass decoded wrong: %q", detail.RClass)
	}
	if detail.XrayIndex == nil || !*detail.XrayIndex {
		t.Errorf("xrayIndex decoded wrong: %v", detail.XrayIndex)
	}
	if detail.AllowAnyHostAuth == nil || !*detail.AllowAnyHostAuth {
		t.Errorf("allowAnyHostAuth decoded wrong: %v", detail.AllowAnyHostAuth)
	}
	if detail.ExternalDependenciesEnabled == nil || !*detail.ExternalDependenciesEnabled {
		t.Errorf("externalDependenciesEnabled decoded wrong: %v", detail.ExternalDependenciesEnabled)
	}
	if len(detail.Repositories) != 2 {
		t.Errorf("member repositories decoded wrong: %v", detail.Repositories)
	}
	if detail.ExcludesPattern != "secret/**" {
		t.Errorf("excludesPattern decoded wrong: %q", detail.ExcludesPattern)
	}
}

// A setting the instance does not report for a repository type must stay
// absent, so a local repository is not reported as one that proxies with
// host authentication turned off.
func TestRepositoryDetailKeepsAbsentSettingsNull(t *testing.T) {
	const payload = `{"key":"example-docker","rclass":"local","packageType":"docker"}`

	var detail repositoryDetailRecord
	if err := json.Unmarshal([]byte(payload), &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.AllowAnyHostAuth != nil || detail.ExternalDependenciesEnabled != nil || detail.XrayIndex != nil {
		t.Errorf("an absent setting decoded as present: %+v", detail)
	}
}

func TestUserRecordDecodes(t *testing.T) {
	const payload = `{
		"username": "example-admin",
		"email": "admin@example.com",
		"admin": true,
		"profile_updatable": false,
		"internal_password_disabled": true,
		"disable_ui_access": false,
		"realm": "internal",
		"status": "enabled",
		"groups": ["platform-admins", "readers"],
		"last_logged_in": "2026-01-02T03:04:05.000Z"
	}`

	var rec userRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if rec.Username != "example-admin" || rec.Email != "admin@example.com" {
		t.Errorf("identity decoded wrong: %+v", rec)
	}
	if !boolValue(rec.Admin) {
		t.Error("admin decoded as false, want true")
	}
	if boolValue(rec.ProfileUpdatable) {
		t.Error("profile_updatable decoded as true, want false")
	}
	if !boolValue(rec.InternalPasswordDisabled) {
		t.Error("internal_password_disabled decoded as false, want true")
	}
	if rec.Realm != "internal" || rec.Status != "enabled" {
		t.Errorf("realm or status decoded wrong: %+v", rec)
	}
	if len(rec.Groups) != 2 {
		t.Errorf("groups decoded wrong: %v", rec.Groups)
	}
	if rec.LastLoggedIn.Time() == nil {
		t.Fatal("last_logged_in decoded as null")
	}
	if got := rec.LastLoggedIn.Time().UTC(); got.Year() != 2026 || got.Month() != time.January {
		t.Errorf("last_logged_in decoded wrong: %v", got)
	}
}

// An account that never signed in reports no timestamp. It must stay null
// rather than becoming the zero time, which would read as a sign-in in year 1.
func TestUserRecordKeepsAbsentLoginNull(t *testing.T) {
	const payload = `{"username":"example","realm":"internal"}`

	var rec userRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.LastLoggedIn.Time() != nil {
		t.Errorf("last_logged_in = %v, want null", rec.LastLoggedIn.Time())
	}
	if boolValue(rec.Admin) {
		t.Error("an absent admin flag decoded as true")
	}
}

func TestGroupRecordDecodes(t *testing.T) {
	const payload = `{
		"name": "platform-admins",
		"description": "publishes platform artifacts",
		"auto_join": false,
		"admin_privileges": true,
		"realm": "ldap",
		"realm_attributes": "ldapGroupName=platform;groupsStrategy=STATIC",
		"external_id": "cn=platform,ou=groups,dc=example,dc=com",
		"members": ["example-admin", "build-account"]
	}`

	var rec groupRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !boolValue(rec.AdminPrivileges) {
		t.Error("admin_privileges decoded as false, want true")
	}
	if boolValue(rec.AutoJoin) {
		t.Error("auto_join decoded as true, want false")
	}
	if rec.Realm != "ldap" || rec.RealmAttributes == "" || rec.ExternalID == "" {
		t.Errorf("realm fields decoded wrong: %+v", rec)
	}
	if len(rec.Members) != 2 {
		t.Errorf("members decoded wrong: %v", rec.Members)
	}
}

func TestTokenRecordDecodes(t *testing.T) {
	const payload = `{"tokens":[
		{"token_id":"1","subject":"jfrt@01ab2c3d/users/build-account","expiry":1798761600,"issued_at":1767225600,"issuer":"jfrt@01ab2c3d","refreshable":true,"description":"ci","scope":"applied-permissions/groups:ci member-of-groups:ci"},
		{"token_id":"2","subject":"jfrt@01ab2c3d/users/example-admin","issued_at":1767225600,"issuer":"jfrt@01ab2c3d","scope":"applied-permissions/admin"}
	]}`

	var response tokenListResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(response.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(response.Tokens))
	}

	first := response.Tokens[0]
	if first.TokenID != "1" || !first.Refreshable || first.Description != "ci" {
		t.Errorf("token decoded wrong: %+v", first)
	}
	if epochTime(first.Expiry) == nil {
		t.Error("expiry decoded as null on a token that expires")
	}

	// A token without an expiry never expires. That must stay null so a policy
	// can find it, rather than becoming 1 January 1970.
	second := response.Tokens[1]
	if second.Expiry != 0 {
		t.Errorf("expiry = %d, want it absent", second.Expiry)
	}
	if epochTime(second.Expiry) != nil {
		t.Error("an absent expiry produced a timestamp")
	}
	if !grantsAdmin(splitScope(second.Scope)) {
		t.Error("an admin-scoped token was not recognised")
	}
}

func TestCleanupPolicyDecodesBothShapes(t *testing.T) {
	const criteria = `{
		"key": "example-retention",
		"description": "keeps the last builds",
		"cronExp": "0 0 2 * * ?",
		"durationInMinutes": 60,
		"enabled": true,
		"skipTrashcan": false,
		"searchCriteria": {
			"packageTypes": ["docker"],
			"repos": ["example-docker"],
			"excludedRepos": ["example-docker-release"],
			"createdBeforeInMonths": 6,
			"lastDownloadedBeforeInMonths": 3,
			"keepLastNVersions": 5
		}
	}`

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "bare array", body: "[" + criteria + "]"},
		{name: "wrapped in an object", body: `{"policies":[` + criteria + `]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records, err := decodeCleanupPolicies([]byte(tc.body))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(records) != 1 {
				t.Fatalf("expected 1 policy, got %d", len(records))
			}

			rec := records[0]
			if rec.Key != "example-retention" || !rec.Enabled || rec.CronExp != "0 0 2 * * ?" {
				t.Errorf("policy decoded wrong: %+v", rec)
			}
			if rec.DurationInMinutes == nil || *rec.DurationInMinutes != 60 {
				t.Errorf("durationInMinutes decoded wrong: %v", rec.DurationInMinutes)
			}
			if len(rec.SearchCriteria.Repos) != 1 || rec.SearchCriteria.Repos[0] != "example-docker" {
				t.Errorf("repos decoded wrong: %v", rec.SearchCriteria.Repos)
			}
			if rec.SearchCriteria.KeepLastNVersions == nil || *rec.SearchCriteria.KeepLastNVersions != 5 {
				t.Errorf("keepLastNVersions decoded wrong: %v", rec.SearchCriteria.KeepLastNVersions)
			}
		})
	}
}

func TestCleanupPolicyKeepsAbsentThresholdsNull(t *testing.T) {
	records, err := decodeCleanupPolicies([]byte(`[{"key":"example","enabled":false,"searchCriteria":{}}]`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	criteria := records[0].SearchCriteria
	if criteria.CreatedBeforeInMonths != nil || criteria.LastDownloadedBeforeInMonths != nil || criteria.KeepLastNVersions != nil {
		t.Errorf("an absent threshold decoded as zero: %+v", criteria)
	}
}

func TestSecurityConfigDecodes(t *testing.T) {
	const payload = `<?xml version="1.0" encoding="UTF-8"?>
<config xmlns="http://artifactory.jfrog.org/xsd/1.0.0">
  <security>
    <anonAccessEnabled>true</anonAccessEnabled>
    <anonAccessToBuildInfosDisabled>true</anonAccessToBuildInfosDisabled>
    <hideUnauthorizedResources>false</hideUnauthorizedResources>
    <userLockPolicy>
      <enabled>true</enabled>
      <loginAttempts>5</loginAttempts>
    </userLockPolicy>
    <passwordSettings>
      <encryptionPolicy>REQUIRED</encryptionPolicy>
      <expirationPolicy>
        <enabled>true</enabled>
        <passwordMaxAge>60</passwordMaxAge>
      </expirationPolicy>
    </passwordSettings>
  </security>
</config>`

	var config securityConfig
	if err := xml.Unmarshal([]byte(payload), &config); err != nil {
		t.Fatalf("decode: %v", err)
	}

	security := config.Security
	if !boolValue(security.AnonAccessEnabled) {
		t.Error("anonAccessEnabled decoded as false, want true")
	}
	if !boolValue(security.AnonAccessToBuildInfosDisabled) {
		t.Error("anonAccessToBuildInfosDisabled decoded as false, want true")
	}
	if boolValue(security.HideUnauthorizedResources) {
		t.Error("hideUnauthorizedResources decoded as true, want false")
	}
	if !boolValue(security.UserLockPolicy.Enabled) || security.UserLockPolicy.LoginAttempts == nil || *security.UserLockPolicy.LoginAttempts != 5 {
		t.Errorf("user lock policy decoded wrong: %+v", security.UserLockPolicy)
	}
	if security.PasswordSettings.EncryptionPolicy != "REQUIRED" {
		t.Errorf("encryptionPolicy decoded wrong: %q", security.PasswordSettings.EncryptionPolicy)
	}
	if !boolValue(security.PasswordSettings.ExpirationPolicy.Enabled) || security.PasswordSettings.ExpirationPolicy.MaxAgeDays == nil {
		t.Errorf("expiration policy decoded wrong: %+v", security.PasswordSettings.ExpirationPolicy)
	}
}

// An instance that leaves anonymous access out of its configuration has it
// off. The absent element must not read as enabled.
func TestSecurityConfigKeepsAbsentSettingsNull(t *testing.T) {
	var config securityConfig
	if err := xml.Unmarshal([]byte(`<config><security></security></config>`), &config); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if config.Security.AnonAccessEnabled != nil {
		t.Errorf("anonAccessEnabled = %v, want it absent", config.Security.AnonAccessEnabled)
	}
	if boolValue(config.Security.AnonAccessEnabled) {
		t.Error("an absent anonAccessEnabled read as enabled")
	}
}

func TestVersionRecordDecodes(t *testing.T) {
	const payload = `{"version":"7.90.10","revision":"79010900","addons":["docker","xray"]}`

	var rec versionRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Version != "7.90.10" || rec.Revision != "79010900" || len(rec.Addons) != 2 {
		t.Errorf("version record decoded wrong: %+v", rec)
	}
}

// The batched configuration endpoint groups repositories under their class.
// A missed class key silently drops every repository in it, and the resources
// then fall back to a per-repository read that an ordinary account cannot make.
func TestRepositoryConfigurationsDecodeEveryClass(t *testing.T) {
	const payload = `{
		"LOCAL": [{"key":"example-docker","rclass":"local","packageType":"docker","blockPushingSchema1":true,"xrayIndex":true}],
		"REMOTE": [{"key":"example-remote","rclass":"remote","packageType":"docker","url":"https://registry.example.com","username":"reader","storeArtifactsLocally":false,"contentSynchronisation":{"enabled":true}}],
		"VIRTUAL": [{"key":"example-virtual","rclass":"virtual","repositories":["example-docker","example-remote"]}],
		"FEDERATED": [{"key":"example-federated","rclass":"federated"}],
		"RELEASE_BUNDLE": [{"key":"release-bundles","rclass":"releasebundle"}],
		"DISTRIBUTION": [{"key":"example-dist","rclass":"distribution"}]
	}`

	var configurations repositoryConfigurations
	if err := json.Unmarshal([]byte(payload), &configurations); err != nil {
		t.Fatalf("decode: %v", err)
	}

	all := configurations.all()
	if len(all) != 6 {
		t.Fatalf("expected 6 repositories across every class, got %d: %+v", len(all), all)
	}

	byKey := map[string]repositoryDetailRecord{}
	for _, rec := range all {
		byKey[rec.Key] = rec
	}
	for _, key := range []string{"example-docker", "example-remote", "example-virtual", "example-federated", "release-bundles", "example-dist"} {
		if _, ok := byKey[key]; !ok {
			t.Errorf("%q was dropped by the class grouping", key)
		}
	}

	local := byKey["example-docker"]
	if local.BlockPushingSchema1 == nil || !*local.BlockPushingSchema1 {
		t.Errorf("blockPushingSchema1 decoded wrong: %v", local.BlockPushingSchema1)
	}

	remote := byKey["example-remote"]
	if remote.Username != "reader" {
		t.Errorf("upstream user name decoded wrong: %q", remote.Username)
	}
	if remote.StoreArtifactsLocally == nil || *remote.StoreArtifactsLocally {
		t.Errorf("storeArtifactsLocally decoded wrong: %v", remote.StoreArtifactsLocally)
	}
	if remote.ContentSynchronisation == nil || remote.ContentSynchronisation.Enabled == nil || !*remote.ContentSynchronisation.Enabled {
		t.Errorf("contentSynchronisation decoded wrong: %+v", remote.ContentSynchronisation)
	}
	if len(byKey["example-virtual"].Repositories) != 2 {
		t.Errorf("virtual members decoded wrong: %v", byKey["example-virtual"].Repositories)
	}
}

// An empty response must produce no repositories rather than a panic, and an
// entry without a key must not become a map entry keyed by the empty string.
func TestRepositoryConfigurationsHandleAnEmptyResponse(t *testing.T) {
	var configurations repositoryConfigurations
	if err := json.Unmarshal([]byte(`{}`), &configurations); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(configurations.all()) != 0 {
		t.Errorf("an empty response produced %d repositories", len(configurations.all()))
	}
}

func TestRepositoryDetailDecodesTheDeliverySettings(t *testing.T) {
	const payload = `{
		"key": "example-docker",
		"rclass": "local",
		"downloadRedirect": true,
		"cdnRedirect": false,
		"archiveBrowsingEnabled": true,
		"priorityResolution": true,
		"signedUrlTtl": 90,
		"xrayDataTtl": 30,
		"projectKey": "example",
		"propertySets": ["artifactory"],
		"notes": "example note"
	}`

	var detail repositoryDetailRecord
	if err := json.Unmarshal([]byte(payload), &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.DownloadRedirect == nil || !*detail.DownloadRedirect {
		t.Errorf("downloadRedirect decoded wrong: %v", detail.DownloadRedirect)
	}
	if detail.CdnRedirect == nil || *detail.CdnRedirect {
		t.Errorf("cdnRedirect decoded wrong: %v", detail.CdnRedirect)
	}
	if detail.SignedURLTTL == nil || *detail.SignedURLTTL != 90 {
		t.Errorf("signedUrlTtl decoded wrong: %v", detail.SignedURLTTL)
	}
	if detail.XrayDataTTL == nil || *detail.XrayDataTTL != 30 {
		t.Errorf("xrayDataTtl decoded wrong: %v", detail.XrayDataTTL)
	}
	if detail.ProjectKey != "example" || detail.Notes != "example note" || len(detail.PropertySets) != 1 {
		t.Errorf("repository metadata decoded wrong: %+v", detail)
	}
}

// A repository resolved by key is built from the configuration response rather
// than from the list, so every field the list would have set must be taken from
// it. A missed field reports empty on a repository that has one.
func TestRepositoryDetailCarriesTheListFields(t *testing.T) {
	const payload = `{"key":"example-docker","rclass":"local","packageType":"docker","description":"platform images","url":""}`

	var detail repositoryDetailRecord
	if err := json.Unmarshal([]byte(payload), &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// These are the fields newArtifactoryRepository sets at creation time. Each
	// one must be readable from the configuration response.
	if detail.Key == "" || detail.RClass == "" || detail.PackageType == "" || detail.Description == "" {
		t.Errorf("a field the list would have set is missing from the detail: %+v", detail)
	}
	if detail.Description != "platform images" {
		t.Errorf("description decoded wrong: %q", detail.Description)
	}
}
