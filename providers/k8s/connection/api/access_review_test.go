// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSubjectAccessAllowed(t *testing.T) {
	client := fake.NewSimpleClientset()
	// The fake apiserver echoes the request back; emulate an authorizer that only
	// allows "get" and records the attributes it received.
	var lastSpec authorizationv1.SubjectAccessReviewSpec
	client.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		sar := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview)
		lastSpec = sar.Spec
		allowed := sar.Spec.ResourceAttributes.Verb == "get"
		sar.Status = authorizationv1.SubjectAccessReviewStatus{Allowed: allowed, Reason: "by test authorizer"}
		return true, sar, nil
	})

	t.Run("allowed verb", func(t *testing.T) {
		allowed, reason, err := subjectAccessAllowed(context.Background(), client,
			"system:serviceaccount:prod:web", "prod", "get", "", "secrets", "")
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, "by test authorizer", reason)
		// the request carried the attributes we passed
		assert.Equal(t, "system:serviceaccount:prod:web", lastSpec.User)
		assert.Equal(t, "prod", lastSpec.ResourceAttributes.Namespace)
		assert.Equal(t, "secrets", lastSpec.ResourceAttributes.Resource)
	})

	t.Run("denied verb", func(t *testing.T) {
		allowed, _, err := subjectAccessAllowed(context.Background(), client,
			"system:serviceaccount:prod:web", "prod", "delete", "", "secrets", "")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
