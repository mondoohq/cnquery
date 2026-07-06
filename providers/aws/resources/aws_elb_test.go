// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTargetGroupBackendTypeShortCircuit verifies that lambdaTargets and
// ipTargets return an empty list (without calling DescribeTargetHealth) for
// target groups whose target type doesn't match, so an instance-type target
// group reports no Lambda or IP backends.
func TestTargetGroupBackendTypeShortCircuit(t *testing.T) {
	t.Run("lambdaTargets empty for instance-type target group", func(t *testing.T) {
		tg := &mqlAwsElbTargetgroup{}
		tg.targetGroup = elbtypes.TargetGroup{TargetType: elbtypes.TargetTypeEnumInstance}
		result, err := tg.lambdaTargets()
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ipTargets empty for instance-type target group", func(t *testing.T) {
		tg := &mqlAwsElbTargetgroup{}
		tg.targetGroup = elbtypes.TargetGroup{TargetType: elbtypes.TargetTypeEnumInstance}
		result, err := tg.ipTargets()
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ipTargets empty for lambda-type target group", func(t *testing.T) {
		tg := &mqlAwsElbTargetgroup{}
		tg.targetGroup = elbtypes.TargetGroup{TargetType: elbtypes.TargetTypeEnumLambda}
		result, err := tg.ipTargets()
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// TestListenerForwardTargetGroupsNoForwardAction verifies that a listener whose
// default action does not forward (no target group ARNs) resolves to an empty
// list without touching the parent load balancer.
func TestListenerForwardTargetGroupsNoForwardAction(t *testing.T) {
	listener := &mqlAwsElbListener{}
	// defaultActionsCache is nil (e.g. a redirect or fixed-response default action)
	result, err := listener.forwardTargetGroups()
	require.NoError(t, err)
	assert.Empty(t, result)
}
