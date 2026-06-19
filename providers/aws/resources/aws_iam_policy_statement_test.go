// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// setString / setMap / setDict build resolved TValue fields so resource methods
// can be exercised without a runtime — GetOrCompute returns the set value
// without invoking the (runtime-backed) compute function.
func setString(v string) plugin.TValue[string] {
	return plugin.TValue[string]{Data: v, State: plugin.StateIsSet}
}

func setMap(v map[string]any) plugin.TValue[map[string]any] {
	return plugin.TValue[map[string]any]{Data: v, State: plugin.StateIsSet}
}

func setDict(v any) plugin.TValue[any] {
	return plugin.TValue[any]{Data: v, State: plugin.StateIsSet}
}

func TestPrincipalsIncludeWildcard(t *testing.T) {
	tests := []struct {
		name       string
		principals map[string]any
		want       bool
	}{
		{"AWS wildcard", map[string]any{"AWS": []any{"*"}}, true},
		{"bare wildcard key", map[string]any{"*": []any{"*"}}, true},
		{"mixed with wildcard", map[string]any{"AWS": []any{"arn:aws:iam::123456789012:root", "*"}}, true},
		{"specific account only", map[string]any{"AWS": []any{"arn:aws:iam::123456789012:root"}}, false},
		{"service principal", map[string]any{"Service": []any{"s3.amazonaws.com"}}, false},
		{"empty", map[string]any{}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, principalsIncludeWildcard(tt.principals))
		})
	}
}

func TestDictIsEmpty(t *testing.T) {
	assert.True(t, dictIsEmpty(nil))
	assert.True(t, dictIsEmpty(map[string]any{}))
	assert.False(t, dictIsEmpty(map[string]any{"StringEquals": map[string]any{"aws:PrincipalOrgID": "o-123"}}))
	// A non-map condition value is treated as "not empty" — we can't prove it
	// scopes nothing, so we don't claim the statement is unconditionally public.
	assert.False(t, dictIsEmpty("unexpected"))
}

func TestPolicyStatementHasPublicPrincipal(t *testing.T) {
	tests := []struct {
		name       string
		effect     string
		principals map[string]any
		want       bool
	}{
		{"allow wildcard", "Allow", map[string]any{"AWS": []any{"*"}}, true},
		{"allow wildcard lowercase effect", "allow", map[string]any{"AWS": []any{"*"}}, true},
		{"deny wildcard", "Deny", map[string]any{"AWS": []any{"*"}}, false},
		{"allow specific", "Allow", map[string]any{"AWS": []any{"arn:aws:iam::123456789012:root"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &mqlAwsIamPolicyStatement{
				Effect:     setString(tt.effect),
				Principals: setMap(tt.principals),
			}
			got, err := stmt.hasPublicPrincipal()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatementsAllowPublic(t *testing.T) {
	wildcard := map[string]any{"AWS": []any{"*"}}
	specific := map[string]any{"AWS": []any{"arn:aws:iam::123456789012:root"}}
	scopingCondition := map[string]any{"StringEquals": map[string]any{"aws:PrincipalOrgID": "o-123"}}

	newStmt := func(effect string, principals map[string]any, conditions any) *mqlAwsIamPolicyStatement {
		return &mqlAwsIamPolicyStatement{
			Effect:     setString(effect),
			Principals: setMap(principals),
			Conditions: setDict(conditions),
		}
	}

	tests := []struct {
		name       string
		statements []any
		want       bool
	}{
		{"public, no conditions", []any{newStmt("Allow", wildcard, nil)}, true},
		{"public but scoped by condition", []any{newStmt("Allow", wildcard, scopingCondition)}, false},
		{"wildcard but denied", []any{newStmt("Deny", wildcard, nil)}, false},
		{"specific principal", []any{newStmt("Allow", specific, nil)}, false},
		{"no statements", []any{}, false},
		{
			"mixed: scoped public then unscoped public",
			[]any{newStmt("Allow", wildcard, scopingCondition), newStmt("Allow", wildcard, nil)},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := statementsAllowPublic(tt.statements)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
