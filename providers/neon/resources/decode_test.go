// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"
	"time"

	"go.mondoo.com/mql/llx"
)

// The API records in this package are decoded by struct tag alone. A mistyped
// tag compiles, lints, and yields a zero value, which for this provider means
// reporting a project as not blocking public connections when it does. These
// tests pin each security-relevant tag against a payload shaped like the
// documented API response.

func TestProjectRecordDecodesSettings(t *testing.T) {
	const payload = `{
		"id": "proj-1",
		"name": "production",
		"platform_id": "aws",
		"region_id": "aws-us-east-1",
		"pg_version": 17,
		"provisioner": "k8s-neonvm",
		"proxy_host": "us-east-1.aws.neon.tech",
		"store_passwords": true,
		"history_retention_seconds": 604800,
		"hipaa_enabled_at": "2026-01-02T03:04:05Z",
		"org_id": "org-1",
		"owner": {"email": "ada@example.com"},
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-02-03T04:05:06Z",
		"compute_last_active_at": "2026-03-04T05:06:07Z",
		"settings": {
			"allowed_ips": {"ips": ["10.0.0.0/8", "192.0.2.1"], "protected_branches_only": true},
			"audit_log_level": "full",
			"block_public_connections": true,
			"block_vpc_connections": false,
			"enable_logical_replication": true,
			"hipaa": true
		}
	}`

	var rec projectRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if rec.ID != "proj-1" || rec.Name != "production" || rec.RegionID != "aws-us-east-1" {
		t.Errorf("identity fields decoded wrong: %+v", rec)
	}
	if rec.PgVersion != 17 {
		t.Errorf("pg_version decoded wrong: %d", rec.PgVersion)
	}
	if !rec.StorePasswords {
		t.Error("store_passwords decoded wrong")
	}
	if rec.HistoryRetentionSeconds == nil || *rec.HistoryRetentionSeconds != 604800 {
		t.Errorf("history retention decoded wrong: %v", rec.HistoryRetentionSeconds)
	}
	if rec.OrgID == nil || *rec.OrgID != "org-1" {
		t.Errorf("org linkage decoded wrong: %v", rec.OrgID)
	}
	if rec.Owner == nil || rec.Owner.Email != "ada@example.com" {
		t.Errorf("owner decoded wrong: %+v", rec.Owner)
	}

	settings := rec.Settings
	if settings == nil {
		t.Fatal("settings did not decode")
	}
	if !boolPtr(settings.BlockPublicConnections) {
		t.Error("block_public_connections decoded wrong")
	}
	if boolPtr(settings.BlockVpcConnections) {
		t.Error("block_vpc_connections decoded wrong")
	}
	if !boolPtr(settings.EnableLogicalReplication) || !boolPtr(settings.Hipaa) {
		t.Errorf("settings booleans decoded wrong: %+v", settings)
	}
	if strPtr(settings.AuditLogLevel) != "full" {
		t.Errorf("audit log level decoded wrong: %v", settings.AuditLogLevel)
	}
	if settings.AllowedIps == nil || settings.AllowedIps.Ips == nil {
		t.Fatal("allowed_ips did not decode")
	}
	if ips := *settings.AllowedIps.Ips; len(ips) != 2 || ips[0] != "10.0.0.0/8" {
		t.Errorf("allowed ips decoded wrong: %v", ips)
	}
	if !boolPtr(settings.AllowedIps.ProtectedBranchesOnly) {
		t.Error("protected_branches_only decoded wrong")
	}

	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := rec.CreatedAt.Time(); got == nil || !got.Equal(want) {
		t.Errorf("created_at decoded wrong: %v", got)
	}
}

// A project with no settings block must report every control as off rather than
// panicking or reporting a control as enabled.
func TestProjectRecordWithoutSettings(t *testing.T) {
	var rec projectRecord
	if err := json.Unmarshal([]byte(`{"id":"proj-1"}`), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Settings != nil {
		t.Fatalf("expected absent settings, got %+v", rec.Settings)
	}
	if rec.OrgID != nil {
		t.Errorf("expected absent org id, got %v", rec.OrgID)
	}
	if rec.HipaaEnabledAt.Time() != nil {
		t.Errorf("expected null hipaa timestamp, got %v", rec.HipaaEnabledAt.Time())
	}
}

// An absent optional boolean means the control was never switched on. Reporting
// it as true would invert the finding.
func TestBoolPtrTreatsAbsentAsOff(t *testing.T) {
	yes, no := true, false
	if boolPtr(nil) {
		t.Error("nil must decode as false")
	}
	if boolPtr(&no) {
		t.Error("false must stay false")
	}
	if !boolPtr(&yes) {
		t.Error("true must stay true")
	}
}

func TestStrPtrTreatsAbsentAsEmpty(t *testing.T) {
	value := "full"
	if strPtr(nil) != "" {
		t.Error("nil must decode as an empty string")
	}
	if strPtr(&value) != "full" {
		t.Error("a set value must be returned unchanged")
	}
}

func TestBranchRecordDecodesProtection(t *testing.T) {
	const payload = `{
		"id": "br-1",
		"project_id": "proj-1",
		"parent_id": "br-parent",
		"name": "main",
		"default": true,
		"protected": true,
		"current_state": "ready",
		"logical_size": 1048576,
		"init_source": "parent-data",
		"restricted_actions": [
			{"name": "delete-rw-endpoint", "reason": "branch is protected"}
		],
		"expires_at": "2026-06-01T00:00:00Z",
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-02-03T04:05:06Z",
		"last_reset_at": "2026-03-04T05:06:07Z"
	}`

	var rec branchRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !rec.Default || !rec.Protected {
		t.Errorf("protection booleans decoded wrong: %+v", rec)
	}
	if rec.ParentID == nil || *rec.ParentID != "br-parent" {
		t.Errorf("parent linkage decoded wrong: %v", rec.ParentID)
	}
	if rec.LogicalSize == nil || *rec.LogicalSize != 1048576 {
		t.Errorf("logical size decoded wrong: %v", rec.LogicalSize)
	}
	if strPtr(rec.InitSource) != "parent-data" {
		t.Errorf("init source decoded wrong: %v", rec.InitSource)
	}
	if len(rec.RestrictedActions) != 1 || rec.RestrictedActions[0].Name != "delete-rw-endpoint" {
		t.Errorf("restricted actions decoded wrong: %+v", rec.RestrictedActions)
	}
	if rec.RestrictedActions[0].Reason == "" {
		t.Error("restricted action reason decoded wrong")
	}
	if rec.ExpiresAt.Time() == nil {
		t.Error("expires_at decoded to nil")
	}
}

// A root branch carries no parent, and a branch with no expiry carries no
// timestamp. Both must stay null rather than becoming an empty reference or the
// zero time.
func TestBranchRecordRootBranch(t *testing.T) {
	var rec branchRecord
	if err := json.Unmarshal([]byte(`{"id":"br-1","name":"main","default":true}`), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.ParentID != nil {
		t.Errorf("expected no parent, got %v", rec.ParentID)
	}
	if rec.ExpiresAt.Time() != nil {
		t.Errorf("expected no expiry, got %v", rec.ExpiresAt.Time())
	}
	if rec.Protected {
		t.Error("expected an unprotected branch")
	}
}

func TestEndpointRecordDecodesExposureFields(t *testing.T) {
	const payload = `{
		"id": "ep-1",
		"project_id": "proj-1",
		"branch_id": "br-1",
		"host": "ep-1.us-east-1.aws.neon.tech",
		"type": "read_write",
		"current_state": "idle",
		"disabled": true,
		"passwordless_access": true,
		"pooler_enabled": true,
		"pooler_mode": "transaction",
		"autoscaling_limit_min_cu": 0.25,
		"autoscaling_limit_max_cu": 4,
		"suspend_timeout_seconds": 300,
		"region_id": "aws-us-east-1",
		"provisioner": "k8s-neonvm",
		"created_at": "2026-01-02T03:04:05Z",
		"last_active": "2026-03-04T05:06:07Z",
		"started_at": "2026-03-04T05:00:00Z",
		"suspended_at": "2026-03-04T06:00:00Z"
	}`

	var rec endpointRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !rec.Disabled || !rec.PasswordlessAccess || !rec.PoolerEnabled {
		t.Errorf("exposure booleans decoded wrong: %+v", rec)
	}
	if rec.BranchID != "br-1" || rec.ProjectID != "proj-1" {
		t.Errorf("linkage decoded wrong: %+v", rec)
	}
	if rec.AutoscalingLimitMinCu != 0.25 || rec.AutoscalingLimitMaxCu != 4 {
		t.Errorf("autoscaling limits decoded wrong: %v / %v", rec.AutoscalingLimitMinCu, rec.AutoscalingLimitMaxCu)
	}
	if rec.SuspendTimeoutSeconds != 300 {
		t.Errorf("suspend timeout decoded wrong: %d", rec.SuspendTimeoutSeconds)
	}
	if rec.PoolerMode != "transaction" || rec.Type != "read_write" {
		t.Errorf("mode fields decoded wrong: %+v", rec)
	}
}

func TestRoleRecordDecodesProtected(t *testing.T) {
	var rec roleRecord
	if err := json.Unmarshal([]byte(`{"name":"neon_superuser","protected":true,"authentication_method":"password","created_at":"2026-01-02T03:04:05Z"}`), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Name != "neon_superuser" || !boolPtr(rec.Protected) {
		t.Errorf("role fields decoded wrong: %+v", rec)
	}
	if rec.AuthenticationMethod != "password" {
		t.Errorf("authentication_method decoded wrong: %q", rec.AuthenticationMethod)
	}

	var plain roleRecord
	if err := json.Unmarshal([]byte(`{"name":"app"}`), &plain); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if boolPtr(plain.Protected) {
		t.Error("an absent protected flag must decode as false")
	}
}

func TestDatabaseRecordDecodesOwner(t *testing.T) {
	const payload = `{
		"id": 42,
		"branch_id": "br-1",
		"name": "neondb",
		"owner_name": "app",
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-02-03T04:05:06Z"
	}`

	var rec databaseRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.ID != 42 || rec.Name != "neondb" || rec.OwnerName != "app" {
		t.Errorf("database fields decoded wrong: %+v", rec)
	}
	if rec.BranchID != "br-1" {
		t.Errorf("branch linkage decoded wrong: %q", rec.BranchID)
	}
}

func TestPermissionRecordDecodesRevocation(t *testing.T) {
	const live = `{"id":"perm-1","granted_to_email":"ada@example.com","granted_at":"2026-01-02T03:04:05Z"}`
	var rec permissionRecord
	if err := json.Unmarshal([]byte(live), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.GrantedToEmail != "ada@example.com" {
		t.Errorf("grantee decoded wrong: %q", rec.GrantedToEmail)
	}
	// A live grant carries no revocation timestamp, which is what separates it
	// from one that has been withdrawn.
	if rec.RevokedAt.Time() != nil {
		t.Errorf("expected a live grant, got revoked at %v", rec.RevokedAt.Time())
	}

	const revoked = `{"id":"perm-2","granted_to_email":"bob@example.com","granted_at":"2026-01-02T03:04:05Z","revoked_at":"2026-04-05T06:07:08Z"}`
	var gone permissionRecord
	if err := json.Unmarshal([]byte(revoked), &gone); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gone.RevokedAt.Time() == nil {
		t.Error("expected a revoked grant")
	}
}

func TestJwksRecordDecodesRoleNames(t *testing.T) {
	const payload = `{
		"id": "jwks-1",
		"project_id": "proj-1",
		"branch_id": "br-1",
		"jwks_url": "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"provider_name": "Clerk",
		"jwt_audience": "authenticated",
		"role_names": ["authenticated", "anon"],
		"created_at": "2026-01-02T03:04:05Z"
	}`

	var rec jwksRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.ProviderName != "Clerk" || rec.JwksURL == "" {
		t.Errorf("provider fields decoded wrong: %+v", rec)
	}
	if strPtr(rec.BranchID) != "br-1" {
		t.Errorf("branch linkage decoded wrong: %v", rec.BranchID)
	}
	if rec.RoleNames == nil || len(*rec.RoleNames) != 2 {
		t.Errorf("role names decoded wrong: %v", rec.RoleNames)
	}
	if strPtr(rec.JwtAudience) != "authenticated" {
		t.Errorf("audience decoded wrong: %v", rec.JwtAudience)
	}
}

func TestOrganizationRecordDecodesHipaaAllowance(t *testing.T) {
	const payload = `{
		"id": "org-1",
		"name": "Acme",
		"handle": "acme",
		"plan": "scale",
		"managed_by": "console",
		"allow_hipaa_projects": true,
		"created_at": "2026-01-02T03:04:05Z",
		"updated_at": "2026-02-03T04:05:06Z"
	}`

	var rec organizationRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Handle != "acme" || rec.Plan != "scale" || rec.ManagedBy != "console" {
		t.Errorf("organization fields decoded wrong: %+v", rec)
	}
	if !boolPtr(rec.AllowHipaaProjects) {
		t.Error("allow_hipaa_projects decoded wrong")
	}
}

// require_mfa is absent on plans that do not offer it. Absent must read as "not
// required" rather than defaulting the other way.
func TestOrganizationRecordDecodesRequireMfa(t *testing.T) {
	var on organizationRecord
	if err := json.Unmarshal([]byte(`{"id":"org-1","require_mfa":true}`), &on); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !boolPtr(on.RequireMfa) {
		t.Error("require_mfa true decoded wrong")
	}

	var off organizationRecord
	if err := json.Unmarshal([]byte(`{"id":"org-1","require_mfa":false}`), &off); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if boolPtr(off.RequireMfa) {
		t.Error("require_mfa false decoded wrong")
	}

	var absent organizationRecord
	if err := json.Unmarshal([]byte(`{"id":"org-1"}`), &absent); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if boolPtr(absent.RequireMfa) {
		t.Error("an absent require_mfa must read as not required")
	}
}

// The member roster nests the role and the email under different objects, so a
// flattened struct tag would silently produce a roster with no email addresses.
func TestMemberWithUserRecordDecodesNesting(t *testing.T) {
	const payload = `{
		"member": {"id":"mem-1","org_id":"org-1","user_id":"user-1","role":"admin","joined_at":"2026-01-02T03:04:05Z"},
		"user": {"email":"ada@example.com"}
	}`

	var rec memberWithUserRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Member.Role != "admin" || rec.Member.ID != "mem-1" {
		t.Errorf("member fields decoded wrong: %+v", rec.Member)
	}
	if rec.User.Email != "ada@example.com" {
		t.Errorf("member email decoded wrong: %q", rec.User.Email)
	}
	if rec.Member.JoinedAt.Time() == nil {
		t.Error("joined_at decoded to nil")
	}
	if rec.User.HasMfa {
		t.Error("has_mfa decoded wrong")
	}

	// has_mfa rides on the nested user object, so a flattened tag would report
	// every member as having no second factor.
	const enrolled = `{"member":{"id":"mem-1","role":"admin"},"user":{"email":"ada@example.com","has_mfa":true}}`
	var mfa memberWithUserRecord
	if err := json.Unmarshal([]byte(enrolled), &mfa); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !mfa.User.HasMfa {
		t.Error("has_mfa true decoded wrong")
	}
}

func TestApiKeyRecordDecodesUsage(t *testing.T) {
	const payload = `{
		"id": 4242,
		"name": "ci",
		"created_at": "2026-01-02T03:04:05Z",
		"created_by": {"id":"user-1","name":"Ada Lovelace"},
		"last_used_at": "2026-03-04T05:06:07Z",
		"last_used_from_addr": "192.0.2.10"
	}`

	var rec apiKeyRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.ID != 4242 || rec.Name != "ci" {
		t.Errorf("key fields decoded wrong: %+v", rec)
	}
	if rec.CreatedBy == nil || rec.CreatedBy.Name != "Ada Lovelace" {
		t.Errorf("creator decoded wrong: %+v", rec.CreatedBy)
	}
	if rec.LastUsedFromAddr != "192.0.2.10" || rec.LastUsedAt.Time() == nil {
		t.Errorf("usage fields decoded wrong: %+v", rec)
	}
}

// A key that has never been used carries neither a timestamp nor an address.
// Both must stay null so an unused key is distinguishable from one used at the
// zero time or from an unknown address.
func TestApiKeyRecordNeverUsed(t *testing.T) {
	var rec apiKeyRecord
	if err := json.Unmarshal([]byte(`{"id":1,"name":"unused","created_at":"2026-01-02T03:04:05Z"}`), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.LastUsedAt.Time() != nil {
		t.Errorf("expected no last-used timestamp, got %v", rec.LastUsedAt.Time())
	}
	if rec.LastUsedFromAddr != "" {
		t.Errorf("expected no last-used address, got %q", rec.LastUsedFromAddr)
	}
}

func TestUserRecordDecodesAuthAccounts(t *testing.T) {
	const payload = `{
		"id": "user-1",
		"email": "ada@example.com",
		"name": "Ada",
		"last_name": "Lovelace",
		"plan": "scale",
		"projects_limit": 10,
		"branches_limit": 20,
		"auth_accounts": [{"provider":"github"},{"provider":"google"}]
	}`

	var rec userRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.ProjectsLimit != 10 || rec.BranchesLimit != 20 {
		t.Errorf("limits decoded wrong: %+v", rec)
	}
	if len(rec.AuthAccounts) != 2 || rec.AuthAccounts[1].Provider != "google" {
		t.Errorf("auth accounts decoded wrong: %+v", rec.AuthAccounts)
	}
}

func TestVpcEndpointRecordDecodes(t *testing.T) {
	var rec vpcEndpointRecord
	if err := json.Unmarshal([]byte(`{"vpc_endpoint_id":"vpce-1","label":"prod"}`), &rec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.VpcEndpointID != "vpce-1" || rec.Label != "prod" {
		t.Errorf("vpc endpoint decoded wrong: %+v", rec)
	}
}

func TestNeonTimeHandlesAbsentAndInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantNil bool
	}{
		{"rfc3339", `"2026-01-02T03:04:05Z"`, false},
		{"fractional seconds", `"2026-01-02T03:04:05.123Z"`, false},
		{"offset", `"2026-01-02T03:04:05+02:00"`, false},
		{"null", `null`, true},
		{"empty string", `""`, true},
		{"unparseable", `"not a timestamp"`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var nt neonTime
			if err := json.Unmarshal([]byte(tc.payload), &nt); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got := nt.Time(); (got == nil) != tc.wantNil {
				t.Fatalf("expected nil=%v, got %v", tc.wantNil, got)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	if got := itoa(42); got != "42" {
		t.Fatalf("expected %q, got %q", "42", got)
	}
	if got := itoa(0); got != "0" {
		t.Fatalf("expected %q, got %q", "0", got)
	}
}

// A plan-gated setting the API omits must report null, not an empty string: a
// policy asking "is audit logging set to full" should see "not answered"
// rather than a value that looks deliberately blank.
func TestOptionalStringReportsAbsentAsNull(t *testing.T) {
	if got := optionalString(nil); got != llx.NilData {
		t.Errorf("nil must map to null, got %v", got)
	}
	empty := ""
	if got := optionalString(&empty); got != llx.NilData {
		t.Errorf("an empty value must map to null, got %v", got)
	}
	value := "full"
	got := optionalString(&value)
	if got == llx.NilData || got.Value != "full" {
		t.Errorf("a set value must be carried through, got %v", got)
	}
}

func TestStrSliceToAny(t *testing.T) {
	got := strSliceToAny([]string{"a", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected widening result: %v", got)
	}
	if empty := strSliceToAny(nil); len(empty) != 0 {
		t.Fatalf("expected an empty slice for nil input, got %v", empty)
	}
}
