// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"testing"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/assert"
)

func TestIsRootAccessNotConfigured(t *testing.T) {
	// Each of these establishes that no centralized root access feature is
	// enabled, so an empty list is the honest answer rather than a lost read.
	t.Run("the account is in no organization", func(t *testing.T) {
		assert.True(t, isRootAccessNotConfigured(&iamtypes.OrganizationNotFoundException{}))
	})

	t.Run("the organization runs in consolidated-billing mode", func(t *testing.T) {
		// The feature cannot be enabled at all in that mode.
		assert.True(t, isRootAccessNotConfigured(&iamtypes.OrganizationNotInAllFeaturesModeException{}))
	})

	t.Run("IAM has no trusted access in the organization", func(t *testing.T) {
		// Trusted access is the prerequisite, so without it nothing is enabled.
		assert.True(t, isRootAccessNotConfigured(&iamtypes.ServiceAccessNotEnabledException{}))
	})

	t.Run("through wrapping", func(t *testing.T) {
		assert.True(t, isRootAccessNotConfigured(
			fmt.Errorf("listing organizations features: %w", &iamtypes.OrganizationNotFoundException{})))
	})

	// These leave the answer unknown. Returning an empty list for them would
	// report centralized root access as switched off -- which is the finding --
	// on an organization we merely failed to read.
	t.Run("a denial is not proof that the feature is off", func(t *testing.T) {
		assert.False(t, isRootAccessNotConfigured(orgDeniedErr()))
	})

	t.Run("a transport error is not proof that the feature is off", func(t *testing.T) {
		assert.False(t, isRootAccessNotConfigured(
			errors.New("dial tcp: lookup iam.amazonaws.com: no such host")))
	})

	t.Run("nil is not a not-configured answer", func(t *testing.T) {
		assert.False(t, isRootAccessNotConfigured(nil))
	})
}
