// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsServiceNotAvailableInRegionError(t *testing.T) {
	t.Run("nil error returns false", func(t *testing.T) {
		assert.False(t, IsServiceNotAvailableInRegionError(nil))
	})

	t.Run("unrelated error returns false", func(t *testing.T) {
		assert.False(t, IsServiceNotAvailableInRegionError(errors.New("some random error")))
	})

	t.Run("no such host", func(t *testing.T) {
		err := errors.New("dial tcp: lookup memorydb.us-west-1.amazonaws.com: no such host")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("UnknownEndpoint", func(t *testing.T) {
		err := errors.New("UnknownEndpoint: could not resolve endpoint")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("could not resolve endpoint", func(t *testing.T) {
		err := errors.New("could not resolve endpoint for region")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("EC2 InvalidAction (Verified Access in unsupported region)", func(t *testing.T) {
		err := fmt.Errorf("operation error EC2: DescribeVerifiedAccessInstances, https response error StatusCode: 400, api error InvalidAction: The action DescribeVerifiedAccessInstances is not valid for this web service.")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("Bedrock UnknownOperationException", func(t *testing.T) {
		err := fmt.Errorf("operation error Bedrock: ListCustomModels, https response error StatusCode: 404, api error UnknownOperationException: Unknown Operation")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("Bedrock ValidationException Unknown operation", func(t *testing.T) {
		err := fmt.Errorf("operation error Bedrock: ListCustomModels, https response error StatusCode: 400, ValidationException: Unknown operation")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("Bedrock Unknown Operation (capitalized)", func(t *testing.T) {
		err := fmt.Errorf("operation error Bedrock: ListCustomModels, https response error StatusCode: 400, ValidationException: Unknown Operation")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})
}
