// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/stretchr/testify/assert"
)

func TestBarbicanRefIsContainer(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{"empty", "", false},
		{"container ref", "https://barbican.example/v1/containers/abc", true},
		{"secret ref", "https://barbican.example/v1/secrets/abc", false},
		{"order ref", "https://barbican.example/v1/orders/abc", false},
		{"container in path segment", "https://example/v1/containers/uuid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, barbicanRefIsContainer(tt.ref))
		})
	}
}

func TestListenerIDsFromLB(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		assert.Empty(t, listenerIDsFromLB(nil))
	})
	t.Run("non-empty IDs are kept, empties skipped", func(t *testing.T) {
		got := listenerIDsFromLB([]listeners.Listener{
			{ID: "l-1"},
			{ID: ""},
			{ID: "l-2"},
		})
		assert.Equal(t, []string{"l-1", "l-2"}, got)
	})
}

func TestPoolIDsFromLB(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		assert.Empty(t, poolIDsFromLB(nil))
	})
	t.Run("non-empty IDs are kept, empties skipped", func(t *testing.T) {
		got := poolIDsFromLB([]pools.Pool{
			{ID: "p-1"},
			{ID: ""},
			{ID: "p-2"},
		})
		assert.Equal(t, []string{"p-1", "p-2"}, got)
	})
}

func TestListenerIDsFromPoolListeners(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		assert.Empty(t, listenerIDsFromPoolListeners(nil))
	})
	t.Run("filters empty IDs", func(t *testing.T) {
		got := listenerIDsFromPoolListeners([]pools.ListenerID{
			{ID: "l-1"},
			{ID: ""},
			{ID: "l-2"},
		})
		assert.Equal(t, []string{"l-1", "l-2"}, got)
	})
}

func TestAdditionalVipsToDict(t *testing.T) {
	t.Run("nil returns empty slice", func(t *testing.T) {
		assert.Empty(t, additionalVipsToDict(nil))
	})
	t.Run("converts vips to dicts", func(t *testing.T) {
		got := additionalVipsToDict([]loadbalancers.AdditionalVip{
			{SubnetID: "s-1", IPAddress: "10.0.0.1"},
			{SubnetID: "s-2", IPAddress: "10.0.0.2"},
		})
		assert.Equal(t, []any{
			map[string]any{"subnet_id": "s-1", "ip_address": "10.0.0.1"},
			map[string]any{"subnet_id": "s-2", "ip_address": "10.0.0.2"},
		}, got)
	})
}

func TestPersistenceToDict(t *testing.T) {
	t.Run("empty type returns nil", func(t *testing.T) {
		assert.Nil(t, persistenceToDict(pools.SessionPersistence{}))
	})
	t.Run("type only", func(t *testing.T) {
		got := persistenceToDict(pools.SessionPersistence{Type: "SOURCE_IP"})
		assert.Equal(t, map[string]any{"type": "SOURCE_IP"}, got)
	})
	t.Run("type and cookie name", func(t *testing.T) {
		got := persistenceToDict(pools.SessionPersistence{Type: "APP_COOKIE", CookieName: "JSESSIONID"})
		assert.Equal(t, map[string]any{"type": "APP_COOKIE", "cookie_name": "JSESSIONID"}, got)
	})
}
