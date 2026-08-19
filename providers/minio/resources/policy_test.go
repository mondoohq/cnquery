// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err, "fixture %s must exist", name)
	return string(data)
}

// TestParseRealBucketPolicies pins the parser against the exact documents a
// real MinIO deployment returned for the three shapes that matter: an
// anonymous read grant, a deny-only policy, and an anonymous grant of every
// action.
func TestParseRealBucketPolicies(t *testing.T) {
	t.Run("anonymous read", func(t *testing.T) {
		policy, err := parsePolicyDocument(fixture(t, "bucket_policy_anonymous_read.json"))
		require.NoError(t, err)
		require.NotNil(t, policy)
		require.Len(t, policy.Statements, 1)

		statement := policy.Statements[0]
		assert.Equal(t, "PublicRead", statement.SID)
		assert.Equal(t, "Allow", statement.Effect)
		assert.Equal(t, []string{"*"}, statement.Principal.Values)
		assert.Equal(t, stringSet{"s3:GetObject"}, statement.Action)
		assert.Equal(t, stringSet{"arn:aws:s3:::public-assets/*"}, statement.Resource)

		assert.True(t, policyGrantsAnonymousAccess(policy), "an Allow to * is anonymous access")
		assert.True(t, policyHasWildcardPrincipal(policy))
		assert.False(t, policyHasWildcardAction(policy), "s3:GetObject carries no wildcard")
		assert.True(t, policyHasWildcardResource(policy), "the resource ends in /*")
		assert.False(t, policyGrantsAdminAccess(policy))
		assert.False(t, policyEnforcesSslOnly(policy))
	})

	t.Run("deny only", func(t *testing.T) {
		policy, err := parsePolicyDocument(fixture(t, "bucket_policy_deny_only.json"))
		require.NoError(t, err)
		require.NotNil(t, policy)
		require.Len(t, policy.Statements, 1)

		// The bucket is NOT anonymously reachable: its only statement names the
		// wildcard principal in order to refuse, not to grant. Reporting this
		// as exposure would be a false alarm on the safest policy shape there
		// is, and reporting the wildcard action as a grant would be worse.
		assert.False(t, policyGrantsAnonymousAccess(policy))
		assert.True(t, policyHasWildcardPrincipal(policy), "a Deny to * still names the wildcard")
		assert.False(t, policyHasWildcardAction(policy), "s3:* here is denied, not allowed")
		assert.False(t, policyHasWildcardResource(policy))
		assert.False(t, policyGrantsAdminAccess(policy))
		assert.True(t, policyEnforcesSslOnly(policy))
	})

	t.Run("anonymous wildcard action", func(t *testing.T) {
		policy, err := parsePolicyDocument(fixture(t, "bucket_policy_wildcard_action.json"))
		require.NoError(t, err)
		require.NotNil(t, policy)

		assert.True(t, policyGrantsAnonymousAccess(policy))
		assert.True(t, policyHasWildcardPrincipal(policy))
		assert.True(t, policyHasWildcardAction(policy))
		assert.True(t, policyHasWildcardResource(policy))
		assert.False(t, policyEnforcesSslOnly(policy))
	})
}

// TestConditionValuesAreListsOnTheWire records that MinIO normalizes a
// condition value written as a bare string into a one-element list when it
// stores the policy. A parser that only accepted the list form would still
// work here, but one that only accepted the string form would not.
func TestConditionValuesAreListsOnTheWire(t *testing.T) {
	raw := fixture(t, "bucket_policy_deny_only.json")
	assert.Contains(t, raw, `"aws:SecureTransport":["false"]`,
		"the server returned the condition value wrapped in a list")

	policy, err := parsePolicyDocument(raw)
	require.NoError(t, err)
	assert.Equal(t, stringSet{"false"}, policy.Statements[0].Condition["Bool"]["aws:SecureTransport"])
}

func TestParseNamedPolicies(t *testing.T) {
	var listed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "list_canned_policies.json")), &listed))

	cases := []struct {
		name           string
		wildcardAction bool
		wildcardRes    bool
		adminAccess    bool
	}{
		// consoleAdmin allows admin:* and kms:* with NO Resource element at
		// all, which is what makes the absent-resource case real rather than
		// hypothetical.
		{"consoleAdmin", true, true, true},
		{"readonly", false, true, false},
		{"readwrite", true, true, false},
		{"writeonly", false, true, false},
		{"diagnostics", false, true, true},
		{"custom-wildcard", true, true, false},
		{"scoped-read", false, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, ok := listed[tc.name]
			require.True(t, ok, "the deployment returned a %s policy", tc.name)

			policy, err := parsePolicyDocument(string(raw))
			require.NoError(t, err)
			require.NotNil(t, policy)

			assert.Equal(t, tc.wildcardAction, policyHasWildcardAction(policy))
			assert.Equal(t, tc.wildcardRes, policyHasWildcardResource(policy))
			assert.Equal(t, tc.adminAccess, policyGrantsAdminAccess(policy))
			// A named policy never carries principals; the attachment supplies
			// them. Reporting one would be a decoding bug.
			for _, statement := range policy.Statements {
				assert.Empty(t, statement.Principal.Values)
			}
		})
	}
}

func TestConsoleAdminHasStatementWithoutResource(t *testing.T) {
	var listed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "list_canned_policies.json")), &listed))

	policy, err := parsePolicyDocument(string(listed["consoleAdmin"]))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(policy.Statements), 2)
	assert.Empty(t, policy.Statements[0].Resource, "admin:* is granted with no Resource element")
	assert.Equal(t, stringSet{"admin:*"}, policy.Statements[0].Action)
}

func TestPolicyShapeVariants(t *testing.T) {
	t.Run("empty document is no policy, not an error", func(t *testing.T) {
		policy, err := parsePolicyDocument("")
		require.NoError(t, err)
		assert.Nil(t, policy)

		policy, err = parsePolicyDocument("   \n ")
		require.NoError(t, err)
		assert.Nil(t, policy)
	})

	t.Run("nil policy answers every predicate with false", func(t *testing.T) {
		assert.False(t, policyGrantsAnonymousAccess(nil))
		assert.False(t, policyHasWildcardPrincipal(nil))
		assert.False(t, policyHasWildcardAction(nil))
		assert.False(t, policyHasWildcardResource(nil))
		assert.False(t, policyGrantsAdminAccess(nil))
		assert.False(t, policyEnforcesSslOnly(nil))
	})

	t.Run("no statements", func(t *testing.T) {
		policy, err := parsePolicyDocument(`{"Version":"2012-10-17","Statement":[]}`)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Empty(t, policy.Statements)
		assert.False(t, policyGrantsAnonymousAccess(policy))
	})

	t.Run("statement written as a single object", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::b/*"}}`)
		require.NoError(t, err)
		require.Len(t, policy.Statements, 1)
		assert.Equal(t, []string{"*"}, policy.Statements[0].Principal.Values)
		assert.Equal(t, stringSet{"s3:GetObject"}, policy.Statements[0].Action)
		assert.True(t, policyGrantsAnonymousAccess(policy))
	})

	t.Run("principal as a bare string", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:*"],"Resource":["*"]}]}`)
		require.NoError(t, err)
		assert.True(t, policyGrantsAnonymousAccess(policy))
	})

	t.Run("principal as an object with a bare string value", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":["s3:*"],"Resource":["*"]}]}`)
		require.NoError(t, err)
		assert.True(t, policyGrantsAnonymousAccess(policy))
	})

	t.Run("named principal is not the wildcard", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":[{"Effect":"Allow","Principal":{"AWS":["arn:aws:iam::1:user/a"]},"Action":["s3:*"],"Resource":["*"]}]}`)
		require.NoError(t, err)
		assert.False(t, policyGrantsAnonymousAccess(policy))
		assert.False(t, policyHasWildcardPrincipal(policy))
		assert.True(t, policyHasWildcardAction(policy))
	})

	t.Run("absent principal is not the wildcard", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":[{"Effect":"Allow","Action":["s3:*"],"Resource":["*"]}]}`)
		require.NoError(t, err)
		assert.False(t, policyGrantsAnonymousAccess(policy))
		assert.False(t, policyHasWildcardPrincipal(policy))
	})

	t.Run("effect matching ignores case", func(t *testing.T) {
		policy, err := parsePolicyDocument(
			`{"Statement":[{"Effect":"allow","Principal":"*","Action":["s3:GetObject"],"Resource":["*"]}]}`)
		require.NoError(t, err)
		assert.True(t, policyGrantsAnonymousAccess(policy))
	})

	t.Run("malformed document is an error, never an empty policy", func(t *testing.T) {
		_, err := parsePolicyDocument(`{"Statement":`)
		require.Error(t, err)

		_, err = parsePolicyDocument(`not json at all`)
		require.Error(t, err)
	})
}

func TestPolicyGrantsAdminAccess(t *testing.T) {
	cases := []struct {
		name     string
		document string
		want     bool
	}{
		{"admin wildcard", `{"Statement":[{"Effect":"Allow","Action":["admin:*"]}]}`, true},
		{"single admin action", `{"Statement":[{"Effect":"Allow","Action":["admin:ServerInfo"]}]}`, true},
		{"bare wildcard covers admin", `{"Statement":[{"Effect":"Allow","Action":["*"]}]}`, true},
		{"denied admin is not granted", `{"Statement":[{"Effect":"Deny","Action":["admin:*"]}]}`, false},
		{"s3 only", `{"Statement":[{"Effect":"Allow","Action":["s3:*"]}]}`, false},
		{"kms only", `{"Statement":[{"Effect":"Allow","Action":["kms:*"]}]}`, false},
		{"no statements", `{"Statement":[]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := parsePolicyDocument(tc.document)
			require.NoError(t, err)
			assert.Equal(t, tc.want, policyGrantsAdminAccess(policy))
		})
	}
}

func TestPolicyEnforcesSslOnly(t *testing.T) {
	cases := []struct {
		name     string
		document string
		want     bool
	}{
		{
			"deny without TLS",
			`{"Statement":[{"Effect":"Deny","Principal":"*","Action":["s3:*"],"Resource":["*"],"Condition":{"Bool":{"aws:SecureTransport":["false"]}}}]}`,
			true,
		},
		{
			"condition value as a bare string",
			`{"Statement":[{"Effect":"Deny","Principal":"*","Action":["s3:*"],"Resource":["*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}`,
			true,
		},
		{
			"allow with the same condition is not enforcement",
			`{"Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:*"],"Resource":["*"],"Condition":{"Bool":{"aws:SecureTransport":["false"]}}}]}`,
			false,
		},
		{
			"deny conditioned on true is not enforcement",
			`{"Statement":[{"Effect":"Deny","Principal":"*","Action":["s3:*"],"Resource":["*"],"Condition":{"Bool":{"aws:SecureTransport":["true"]}}}]}`,
			false,
		},
		{
			"no condition",
			`{"Statement":[{"Effect":"Deny","Principal":"*","Action":["s3:*"],"Resource":["*"]}]}`,
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := parsePolicyDocument(tc.document)
			require.NoError(t, err)
			assert.Equal(t, tc.want, policyEnforcesSslOnly(policy))
		})
	}
}

func TestConditionsToDict(t *testing.T) {
	assert.Nil(t, conditionsToDict(nil))
	assert.Nil(t, conditionsToDict(map[string]map[string]stringSet{}))

	out := conditionsToDict(map[string]map[string]stringSet{
		"Bool": {"aws:SecureTransport": stringSet{"false"}},
	})
	require.Contains(t, out, "Bool")
	inner, ok := out["Bool"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"false"}, inner["aws:SecureTransport"])
}

func TestSplitPolicyNames(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"readonly", []string{"readonly"}},
		{"readonly,readwrite", []string{"readonly", "readwrite"}},
		{" readonly , readwrite ", []string{"readonly", "readwrite"}},
		{",", []string{}},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, splitPolicyNames(tc.in), "input %q", tc.in)
	}
}
