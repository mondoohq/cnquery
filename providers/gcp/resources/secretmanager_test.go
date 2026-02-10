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

func TestExtractCustomerManagedEncryption(t *testing.T) {
	t.Run("top-level CMEK", func(t *testing.T) {
		s := &secretmanagerpb.Secret{
			CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
				KmsKeyName: "projects/p/locations/global/keyRings/kr/cryptoKeys/key1",
			},
		}
		cme := extractCustomerManagedEncryption(s)
		require.NotNil(t, cme)
		assert.Equal(t, "projects/p/locations/global/keyRings/kr/cryptoKeys/key1", cme.KmsKeyName)
	})

	t.Run("automatic replication CMEK", func(t *testing.T) {
		s := &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{
						CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
							KmsKeyName: "projects/p/locations/global/keyRings/kr/cryptoKeys/key2",
						},
					},
				},
			},
		}
		cme := extractCustomerManagedEncryption(s)
		require.NotNil(t, cme)
		assert.Equal(t, "projects/p/locations/global/keyRings/kr/cryptoKeys/key2", cme.KmsKeyName)
	})

	t.Run("user-managed replication CMEK", func(t *testing.T) {
		s := &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_UserManaged_{
					UserManaged: &secretmanagerpb.Replication_UserManaged{
						Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
							{
								Location: "us-east1",
								CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
									KmsKeyName: "projects/p/locations/us-east1/keyRings/kr/cryptoKeys/key3",
								},
							},
							{Location: "europe-west1"},
						},
					},
				},
			},
		}
		cme := extractCustomerManagedEncryption(s)
		require.NotNil(t, cme)
		assert.Equal(t, "projects/p/locations/us-east1/keyRings/kr/cryptoKeys/key3", cme.KmsKeyName)
	})

	t.Run("no CMEK anywhere", func(t *testing.T) {
		s := &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		}
		assert.Nil(t, extractCustomerManagedEncryption(s))
	})

	t.Run("nil replication", func(t *testing.T) {
		s := &secretmanagerpb.Secret{}
		assert.Nil(t, extractCustomerManagedEncryption(s))
	})

	t.Run("top-level takes precedence over replication", func(t *testing.T) {
		s := &secretmanagerpb.Secret{
			CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
				KmsKeyName: "projects/p/locations/global/keyRings/kr/cryptoKeys/top-level",
			},
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{
						CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
							KmsKeyName: "projects/p/locations/global/keyRings/kr/cryptoKeys/auto",
						},
					},
				},
			},
		}
		cme := extractCustomerManagedEncryption(s)
		require.NotNil(t, cme)
		assert.Equal(t, "projects/p/locations/global/keyRings/kr/cryptoKeys/top-level", cme.KmsKeyName)
	})
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
