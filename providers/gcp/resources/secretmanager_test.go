// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSecretReplicationToDict_Automatic(t *testing.T) {
	r := &secretmanagerpb.Replication{
		Replication: &secretmanagerpb.Replication_Automatic_{
			Automatic: &secretmanagerpb.Replication_Automatic{},
		},
	}

	result, err := secretReplicationToDict(r)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "AUTOMATIC", result["type"])
	assert.Nil(t, result["replicas"])
}

func TestSecretReplicationToDict_UserManaged(t *testing.T) {
	r := &secretmanagerpb.Replication{
		Replication: &secretmanagerpb.Replication_UserManaged_{
			UserManaged: &secretmanagerpb.Replication_UserManaged{
				Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
					{Location: "us-east1"},
					{Location: "europe-west1"},
				},
			},
		},
	}

	result, err := secretReplicationToDict(r)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "USER_MANAGED", result["type"])
	replicas, ok := result["replicas"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"us-east1", "europe-west1"}, replicas)
}

func TestSecretReplicationToDict_Nil(t *testing.T) {
	r := &secretmanagerpb.Replication{}

	result, err := secretReplicationToDict(r)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestTimestampToString(t *testing.T) {
	t.Run("nil timestamp", func(t *testing.T) {
		assert.Equal(t, "", timestampToString(nil))
	})

	t.Run("valid timestamp", func(t *testing.T) {
		ts := timestamppb.New(time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC))
		result := timestampToString(ts)
		assert.Equal(t, "2024-06-15T10:30:00Z", result)
	})
}

func TestDurationToString(t *testing.T) {
	t.Run("nil duration", func(t *testing.T) {
		assert.Equal(t, "", durationToString(nil))
	})

	t.Run("valid duration", func(t *testing.T) {
		d := &durationpb.Duration{Seconds: 86400}
		assert.Equal(t, "86400s", durationToString(d))
	})

	t.Run("zero duration", func(t *testing.T) {
		d := &durationpb.Duration{Seconds: 0}
		assert.Equal(t, "0s", durationToString(d))
	})
}
