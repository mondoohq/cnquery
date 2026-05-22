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

	t.Run("retry exhaustion with request send failure (bedrock-agent us-west-1 5xx)", func(t *testing.T) {
		// us-west-1 returns HTTP 500 for ListFlows on every retry; after the
		// SDK exhausts attempts the wrapped error contains both phrases.
		err := fmt.Errorf("operation error Bedrock Agent: ListFlows, exceeded maximum number of attempts, 3, https response error StatusCode: 0, RequestID: , request send failed, Get \"https://bedrock-agent.us-west-1.amazonaws.com/flows/\": GET https://bedrock-agent.us-west-1.amazonaws.com/flows/ giving up after 3 attempt(s)")
		assert.True(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("retry exhaustion WITHOUT send failure is NOT swallowed (throttling, real 5xx)", func(t *testing.T) {
		// Throttling/transient 5xx errors that successfully reach the server
		// must propagate up to the caller — the helper should not match
		// "exceeded maximum number of attempts" by itself.
		err := fmt.Errorf("operation error S3: GetObject, exceeded maximum number of attempts, 3, https response error StatusCode: 503, RequestID: ABC, api error SlowDown: Please reduce your request rate.")
		assert.False(t, IsServiceNotAvailableInRegionError(err))
	})

	t.Run("plain request send failed (no retry exhaustion) does NOT match", func(t *testing.T) {
		// A single transient network error without retry exhaustion is also
		// not enough — both phrases must be present together.
		err := fmt.Errorf("operation error EC2: DescribeInstances, https response error StatusCode: 0, request send failed, Get \"https://ec2.amazonaws.com/\": net/http: TLS handshake timeout")
		assert.False(t, IsServiceNotAvailableInRegionError(err))
	})
}
