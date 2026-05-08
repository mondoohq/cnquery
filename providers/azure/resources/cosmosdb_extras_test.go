// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	cosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/stretchr/testify/assert"
)

func TestCosmosAccountResourceGroup(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		wantSub     string
		wantRG      string
		wantAccount string
		wantErr     bool
	}{
		{
			name:        "well-formed account id",
			id:          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg/providers/Microsoft.DocumentDB/databaseAccounts/myacct",
			wantSub:     "00000000-0000-0000-0000-000000000000",
			wantRG:      "my-rg",
			wantAccount: "myacct",
		},
		{
			name:    "missing databaseAccounts segment",
			id:      "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg/providers/Microsoft.DocumentDB",
			wantErr: true,
		},
		{
			name:    "malformed id",
			id:      "not-an-arm-id",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub, rg, acct, err := cosmosAccountResourceGroup(tc.id)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantSub, sub)
			assert.Equal(t, tc.wantRG, rg)
			assert.Equal(t, tc.wantAccount, acct)
		})
	}
}

func TestThroughputFromResource(t *testing.T) {
	t.Run("nil props returns zero", func(t *testing.T) {
		manual, auto, autoEnabled, shared := throughputFromResource(nil)
		assert.Equal(t, int32(0), manual)
		assert.Equal(t, int32(0), auto)
		assert.False(t, autoEnabled)
		assert.False(t, shared)
	})

	t.Run("manual throughput only", func(t *testing.T) {
		tp := int32(400)
		props := &cosmos.ThroughputSettingsGetProperties{
			Resource: &cosmos.ThroughputSettingsGetPropertiesResource{
				Throughput: &tp,
			},
		}
		manual, auto, autoEnabled, shared := throughputFromResource(props)
		assert.Equal(t, int32(400), manual)
		assert.Equal(t, int32(0), auto)
		assert.False(t, autoEnabled)
		assert.False(t, shared)
	})

	t.Run("autoscale only", func(t *testing.T) {
		max := int32(4000)
		props := &cosmos.ThroughputSettingsGetProperties{
			Resource: &cosmos.ThroughputSettingsGetPropertiesResource{
				AutoscaleSettings: &cosmos.AutoscaleSettingsResource{
					MaxThroughput: &max,
				},
			},
		}
		manual, auto, autoEnabled, shared := throughputFromResource(props)
		assert.Equal(t, int32(0), manual)
		assert.Equal(t, int32(4000), auto)
		assert.True(t, autoEnabled)
		assert.False(t, shared)
	})
}

func TestIsCosmosNotFoundError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.False(t, isCosmosNotFoundError(nil))
	})
	t.Run("non-azcore error", func(t *testing.T) {
		assert.False(t, isCosmosNotFoundError(errors.New("boom")))
	})
	t.Run("404 azcore.ResponseError", func(t *testing.T) {
		assert.True(t, isCosmosNotFoundError(&azcore.ResponseError{StatusCode: http.StatusNotFound}))
	})
	t.Run("403 not treated as not-found", func(t *testing.T) {
		assert.False(t, isCosmosNotFoundError(&azcore.ResponseError{StatusCode: http.StatusForbidden}))
	})
}

func TestIsCosmosForbiddenError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.False(t, isCosmosForbiddenError(nil))
	})
	t.Run("403 azcore.ResponseError", func(t *testing.T) {
		assert.True(t, isCosmosForbiddenError(&azcore.ResponseError{StatusCode: http.StatusForbidden}))
	})
	t.Run("404 not treated as forbidden", func(t *testing.T) {
		assert.False(t, isCosmosForbiddenError(&azcore.ResponseError{StatusCode: http.StatusNotFound}))
	})
}
