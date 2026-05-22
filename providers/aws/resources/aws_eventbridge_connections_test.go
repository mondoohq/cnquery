// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The EventBridge typed-ref methods (apiDestination.connection,
// archive.eventSource, replay.eventSource, replay.destination) must mark the
// reference as set+null when the source ARN field is empty — otherwise the
// runtime panics on the unresolved field rather than treating it as a clean
// absence. These tests pin the empty-arn contract without needing a runtime
// mock; the non-empty path is exercised by interactive verification.

func TestApiDestinationConnectionNullWhenArnEmpty(t *testing.T) {
	a := &mqlAwsEventbridgeApiDestination{}
	got, err := a.connection()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, a.Connection.IsNull())
	assert.True(t, a.Connection.IsSet())
}

func TestArchiveEventSourceNullWhenArnEmpty(t *testing.T) {
	a := &mqlAwsEventbridgeArchive{}
	got, err := a.eventSource()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, a.EventSource.IsNull())
	assert.True(t, a.EventSource.IsSet())
}

func TestReplayEventSourceNullWhenArnEmpty(t *testing.T) {
	r := &mqlAwsEventbridgeReplay{}
	got, err := r.eventSource()
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.True(t, r.EventSource.IsNull())
	assert.True(t, r.EventSource.IsSet())
}
