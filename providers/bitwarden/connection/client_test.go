// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestFlexEnumUnmarshalJSON(t *testing.T) {
	t.Run("integer ordinal", func(t *testing.T) {
		var f flexEnum
		if err := json.Unmarshal([]byte("2"), &f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.asInt == nil || *f.asInt != 2 {
			t.Fatalf("asInt = %v, want 2", f.asInt)
		}
		if f.asString != nil {
			t.Fatalf("asString = %v, want nil", f.asString)
		}
	})

	t.Run("string name", func(t *testing.T) {
		var f flexEnum
		if err := json.Unmarshal([]byte(`"Owner"`), &f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.asString == nil || *f.asString != "Owner" {
			t.Fatalf("asString = %v, want Owner", f.asString)
		}
		if f.asInt != nil {
			t.Fatalf("asInt = %v, want nil", f.asInt)
		}
	})

	t.Run("null leaves both unset", func(t *testing.T) {
		var f flexEnum
		if err := json.Unmarshal([]byte("null"), &f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.asInt != nil || f.asString != nil {
			t.Fatalf("expected both nil, got asInt=%v asString=%v", f.asInt, f.asString)
		}
	})

	t.Run("invalid input errors", func(t *testing.T) {
		var f flexEnum
		if err := json.Unmarshal([]byte("{}"), &f); err == nil {
			t.Fatal("expected error decoding an object into flexEnum, got nil")
		}
	})

	t.Run("negative ordinal decodes", func(t *testing.T) {
		var f flexEnum
		if err := json.Unmarshal([]byte("-1"), &f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.asInt == nil || *f.asInt != -1 {
			t.Fatalf("asInt = %v, want -1", f.asInt)
		}
	})
}

// intEnum / strEnum build a flexEnum in the wire form the API would send.
func intEnum(i int) flexEnum    { return flexEnum{asInt: &i} }
func strEnum(s string) flexEnum { return flexEnum{asString: &s} }

func TestPolicyTypeName(t *testing.T) {
	tests := []struct {
		name string
		in   flexEnum
		want string
	}{
		{"ordinal 0", intEnum(0), "twoFactorAuthentication"},
		{"ordinal 8", intEnum(8), "resetPassword"},
		{"last ordinal", intEnum(len(policyTypeNames) - 1), policyTypeNames[len(policyTypeNames)-1]},
		{"out-of-range ordinal", intEnum(len(policyTypeNames)), "unknown(" + strconv.Itoa(len(policyTypeNames)) + ")"},
		{"negative ordinal", intEnum(-5), "unknown(-5)"},
		{"pascalcase string", strEnum("MasterPassword"), "masterPassword"},
		{"already camel string", strEnum("passwordGenerator"), "passwordGenerator"},
		{"empty string", strEnum(""), ""},
		{"unset", flexEnum{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PolicyTypeName(tc.in); got != tc.want {
				t.Fatalf("PolicyTypeName(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMemberRoleName(t *testing.T) {
	tests := []struct {
		name string
		in   flexEnum
		want string
	}{
		{"owner", intEnum(0), "owner"},
		{"admin", intEnum(1), "admin"},
		{"user", intEnum(2), "user"},
		{"manager", intEnum(3), "manager"},
		{"custom", intEnum(4), "custom"},
		{"out-of-range", intEnum(5), "unknown(5)"},
		{"string owner", strEnum("Owner"), "owner"},
		{"string custom", strEnum("Custom"), "custom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MemberRoleName(tc.in); got != tc.want {
				t.Fatalf("MemberRoleName(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMemberStatusName(t *testing.T) {
	tests := []struct {
		name string
		in   flexEnum
		want string
	}{
		{"revoked (-1)", intEnum(-1), "revoked"},
		{"invited", intEnum(0), "invited"},
		{"accepted", intEnum(1), "accepted"},
		{"confirmed", intEnum(2), "confirmed"},
		{"staged", intEnum(3), "staged"},
		{"out-of-range", intEnum(4), "unknown(4)"},
		{"string confirmed", strEnum("Confirmed"), "confirmed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MemberStatusName(tc.in); got != tc.want {
				t.Fatalf("MemberStatusName(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestResolveEnumStringBranch pins the expected wire format for string-valued
// enums. The API sends PascalCase names, and lowercasing only the first rune
// turns them into the camelCase names the .lr schema documents, including
// multi-word names like "TwoFactorAuthentication" -> "twoFactorAuthentication".
// A blanket strings.ToLower would flatten those multi-word names incorrectly
// ("twofactorauthentication"), which is why the first-rune transform is used.
// This test fails if that behavior regresses, and documents that an all-caps
// wire value ("OWNER") is NOT normalized, so any future API change to all-caps
// values is caught here rather than silently mislabeled.
func TestResolveEnumStringBranch(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"TwoFactorAuthentication", "twoFactorAuthentication"},
		{"MasterPassword", "masterPassword"},
		{"Owner", "owner"},
		{"passwordGenerator", "passwordGenerator"},
		{"", ""},
		// documented limitation: all-caps is not folded to lower-case
		{"OWNER", "oWNER"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := resolveEnum(strEnum(tc.in), memberRoleNames); got != tc.want {
				t.Fatalf("resolveEnum(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMemberDecode(t *testing.T) {
	payload := `{
		"id": "member-1",
		"userId": "user-1",
		"name": "Jane Doe",
		"email": "jane@example.com",
		"type": 0,
		"status": 2,
		"twoFactorEnabled": true,
		"resetPasswordEnrolled": false,
		"externalId": null,
		"collections": [
			{"id": "col-1", "readOnly": true, "hidePasswords": false, "manage": false}
		]
	}`
	var m Member
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Id != "member-1" {
		t.Fatalf("Id = %q, want member-1", m.Id)
	}
	if m.UserId == nil || *m.UserId != "user-1" {
		t.Fatalf("UserId = %v, want user-1", m.UserId)
	}
	if m.ExternalId != nil {
		t.Fatalf("ExternalId = %v, want nil", m.ExternalId)
	}
	if !m.TwoFactorEnabled {
		t.Fatal("TwoFactorEnabled = false, want true")
	}
	if got := MemberRoleName(m.Type); got != "owner" {
		t.Fatalf("role = %q, want owner", got)
	}
	if got := MemberStatusName(m.Status); got != "confirmed" {
		t.Fatalf("status = %q, want confirmed", got)
	}
	if len(m.Collections) != 1 || m.Collections[0].Id != "col-1" || !m.Collections[0].ReadOnly {
		t.Fatalf("collections decode mismatch: %+v", m.Collections)
	}
}

func TestPolicyDecode(t *testing.T) {
	payload := `{
		"id": "policy-1",
		"type": 0,
		"enabled": true,
		"data": {"minComplexity": 3}
	}`
	var p Policy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Id != "policy-1" || !p.Enabled {
		t.Fatalf("policy scalar decode mismatch: %+v", p)
	}
	if got := PolicyTypeName(p.Type); got != "twoFactorAuthentication" {
		t.Fatalf("policy type = %q, want twoFactorAuthentication", got)
	}
	if p.Data["minComplexity"] != float64(3) {
		t.Fatalf("Data = %v, want minComplexity=3", p.Data)
	}
}

func TestGroupAndCollectionDecode(t *testing.T) {
	groupPayload := `{
		"id": "group-1",
		"name": "Engineering",
		"externalId": "ext-1",
		"collections": [{"id": "col-1", "readOnly": false, "hidePasswords": true, "manage": true}]
	}`
	var g Group
	if err := json.Unmarshal([]byte(groupPayload), &g); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name != "Engineering" || g.ExternalId == nil || *g.ExternalId != "ext-1" {
		t.Fatalf("group decode mismatch: %+v", g)
	}
	if len(g.Collections) != 1 || !g.Collections[0].HidePasswords || !g.Collections[0].Manage {
		t.Fatalf("group collection grants mismatch: %+v", g.Collections)
	}

	collectionPayload := `{
		"id": "col-1",
		"name": "Shared",
		"externalId": null,
		"groups": [{"id": "group-1", "readOnly": true, "hidePasswords": false, "manage": false}]
	}`
	var c Collection
	if err := json.Unmarshal([]byte(collectionPayload), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Shared" || c.ExternalId != nil {
		t.Fatalf("collection decode mismatch: %+v", c)
	}
	if len(c.Groups) != 1 || c.Groups[0].Id != "group-1" || !c.Groups[0].ReadOnly {
		t.Fatalf("collection group grants mismatch: %+v", c.Groups)
	}
}

func TestListResponseDecode(t *testing.T) {
	payload := `{"object":"list","data":[{"id":"p1","type":1,"enabled":false,"data":null}],"continuationToken":null}`
	var out listResponse[Policy]
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Object != "list" || len(out.Data) != 1 || out.Data[0].Id != "p1" {
		t.Fatalf("list envelope decode mismatch: %+v", out)
	}
	if out.ContinuationToken != nil {
		t.Fatalf("ContinuationToken = %v, want nil", out.ContinuationToken)
	}
}
