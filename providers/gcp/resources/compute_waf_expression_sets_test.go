// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	compute "google.golang.org/api/compute/v1"
)

func TestWafExpressionSetArgs(t *testing.T) {
	args, err := wafExpressionSetArgs("proj-1", &compute.WafExpressionSet{
		Id:      "sqli-v33-stable",
		Aliases: []string{"sqli-stable"},
		Expressions: []*compute.WafExpressionSetExpression{
			{Id: "owasp-crs-v030301-id942100-sqli", Sensitivity: 1},
			{Id: "owasp-crs-v030301-id942110-sqli", Sensitivity: 3},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "sqli-v33-stable", args["id"].Value)
	assert.Equal(t, []interface{}{"sqli-stable"}, args["aliases"].Value)

	exprs, ok := args["expressions"].Value.([]interface{})
	require.True(t, ok)
	require.Len(t, exprs, 2)

	first, ok := exprs[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "owasp-crs-v030301-id942100-sqli", first["id"])
	// Sensitivity is the ModSecurity paranoia level and is the reason this
	// resource is worth reading: a policy running a set at level 1 catches far
	// less than the same set at level 3. Dropping it would leave the
	// expressions indistinguishable from a bare ID list.
	assert.EqualValues(t, 1, first["sensitivity"])

	second := exprs[1].(map[string]interface{})
	assert.EqualValues(t, 3, second["sensitivity"])
}

// The expression sets are Google-maintained and identical across projects, so
// the set ID alone is not a unique cache key. Scanning two projects in one run
// would make the second project's sets resolve to the first project's.
func TestWafExpressionSetArgsScopesTheCacheKeyToTheProject(t *testing.T) {
	a, err := wafExpressionSetArgs("proj-1", &compute.WafExpressionSet{Id: "xss-v33-stable"})
	require.NoError(t, err)
	b, err := wafExpressionSetArgs("proj-2", &compute.WafExpressionSet{Id: "xss-v33-stable"})
	require.NoError(t, err)

	assert.NotEqual(t, a["__id"].Value, b["__id"].Value)
}

// A set with no aliases or expressions must map to empty lists rather than
// null, so a policy counting expressions sees zero instead of an unread field.
func TestWafExpressionSetArgsEmptyCollections(t *testing.T) {
	args, err := wafExpressionSetArgs("proj-1", &compute.WafExpressionSet{Id: "empty-set"})
	require.NoError(t, err)

	assert.Equal(t, []interface{}{}, args["aliases"].Value)
	assert.Equal(t, []interface{}{}, args["expressions"].Value)
}

// Every level of the response is optional in the API. A nil anywhere in the
// chain must read as "no sets", not panic the provider and take the scan with
// it.
func TestWafExpressionSetsFromResponseToleratesNils(t *testing.T) {
	assert.Nil(t, wafExpressionSetsFromResponse(nil))
	assert.Nil(t, wafExpressionSetsFromResponse(&compute.SecurityPoliciesListPreconfiguredExpressionSetsResponse{}))
	assert.Nil(t, wafExpressionSetsFromResponse(&compute.SecurityPoliciesListPreconfiguredExpressionSetsResponse{
		PreconfiguredExpressionSets: &compute.SecurityPoliciesWafConfig{},
	}))

	sets := wafExpressionSetsFromResponse(&compute.SecurityPoliciesListPreconfiguredExpressionSetsResponse{
		PreconfiguredExpressionSets: &compute.SecurityPoliciesWafConfig{
			WafRules: &compute.PreconfiguredWafSet{
				ExpressionSets: []*compute.WafExpressionSet{{Id: "sqli-v33-stable"}},
			},
		},
	})
	require.Len(t, sets, 1)
	assert.Equal(t, "sqli-v33-stable", sets[0].Id)
}
