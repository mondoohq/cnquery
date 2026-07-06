// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringValue(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", stringValue(nil))
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", stringValue(&s))
	})

	t.Run("empty string pointer returns empty string", func(t *testing.T) {
		s := ""
		assert.Equal(t, "", stringValue(&s))
	})
}

func TestBoolValue(t *testing.T) {
	t.Run("nil returns false", func(t *testing.T) {
		assert.False(t, boolValue(nil))
	})

	t.Run("true pointer returns true", func(t *testing.T) {
		b := true
		assert.True(t, boolValue(&b))
	})

	t.Run("false pointer returns false", func(t *testing.T) {
		b := false
		assert.False(t, boolValue(&b))
	})
}

func TestInt64Value(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		assert.Equal(t, int64(0), int64Value(nil))
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		i := int64(42)
		assert.Equal(t, int64(42), int64Value(&i))
	})

	t.Run("zero pointer returns 0", func(t *testing.T) {
		i := int64(0)
		assert.Equal(t, int64(0), int64Value(&i))
	})
}

func TestIntValue(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		assert.Equal(t, int64(0), intValue(nil))
	})

	t.Run("non-nil returns value as int64", func(t *testing.T) {
		i := 42
		assert.Equal(t, int64(42), intValue(&i))
	})
}

func TestOciResourceTypeFromOCID(t *testing.T) {
	tests := []struct {
		name string
		ocid string
		want string
	}{
		{"internet gateway", "ocid1.internetgateway.oc1.iad.aaaaaaaaexample", "internetgateway"},
		{"drg", "ocid1.drg.oc1.phx.aaaaaaaaexample", "drg"},
		{"private ip", "ocid1.privateip.oc1.iad.aaaaaaaaexample", "privateip"},
		{"empty string", "", ""},
		{"no dots", "ocid1", ""},
		{"only ocid1 prefix", "ocid1.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociResourceTypeFromOCID(tt.ocid))
		})
	}
}

func TestOciRouteTargetType(t *testing.T) {
	tests := []struct {
		name string
		ocid string
		want string
	}{
		{"internet gateway", "ocid1.internetgateway.oc1.iad.aaaa", "INTERNET_GATEWAY"},
		{"nat gateway", "ocid1.natgateway.oc1.iad.aaaa", "NAT_GATEWAY"},
		{"service gateway", "ocid1.servicegateway.oc1.iad.aaaa", "SERVICE_GATEWAY"},
		{"drg", "ocid1.drg.oc1.iad.aaaa", "DRG"},
		{"local peering gateway", "ocid1.localpeeringgateway.oc1.iad.aaaa", "LOCAL_PEERING_GATEWAY"},
		{"private ip", "ocid1.privateip.oc1.iad.aaaa", "PRIVATE_IP"},
		{"unknown type falls back to uppercased raw type", "ocid1.dynamicroutingentity.oc1.iad.aaaa", "DYNAMICROUTINGENTITY"},
		{"malformed OCID", "not-an-ocid", ""},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ociRouteTargetType(tt.ocid))
		})
	}
}
