// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// requireDictSerializes runs a dict through the exact llx conversion path a
// queried `dict` field takes. The dict-to-primitive converter only accepts
// bool/int64/float64/string/[]any/map[string]any; a raw `int` or `time.Time`
// errors here, which is how the load balancer health-check and deprecation
// dicts silently broke. This is the regression guard for those bugs.
func requireDictSerializes(t *testing.T, d map[string]any) {
	t.Helper()
	res := llx.DictData(d).Result()
	require.Empty(t, res.Error, "dict must serialize through the llx dict converter")
}

// requireDictArraySerializes is requireDictSerializes for a `[]dict` field.
func requireDictArraySerializes(t *testing.T, a []any) {
	t.Helper()
	res := llx.ArrayData(a, types.Dict).Result()
	require.Empty(t, res.Error, "[]dict must serialize through the llx dict converter")
}

func TestDeprecationDict(t *testing.T) {
	t.Run("nil is an empty dict and serializes", func(t *testing.T) {
		d := deprecationDict(nil)
		assert.Equal(t, map[string]any{}, d)
		requireDictSerializes(t, d)
	})

	t.Run("deprecated uses RFC3339 strings and serializes", func(t *testing.T) {
		announced := time.Date(2025, 9, 24, 12, 0, 0, 0, time.UTC)
		unavailable := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
		d := deprecationDict(&hcloud.DeprecationInfo{
			Announced:        announced,
			UnavailableAfter: unavailable,
		})
		// Values must be strings, not time.Time — a time.Time fails serialization.
		assert.Equal(t, "2025-09-24T12:00:00Z", d["announced"])
		assert.Equal(t, "2026-03-24T12:00:00Z", d["unavailableAfter"])
		requireDictSerializes(t, d)
	})

	t.Run("zero timestamps are omitted", func(t *testing.T) {
		d := deprecationDict(&hcloud.DeprecationInfo{})
		assert.NotContains(t, d, "announced")
		assert.NotContains(t, d, "unavailableAfter")
		requireDictSerializes(t, d)
	})
}

func TestLoadBalancerHealthCheckDict(t *testing.T) {
	t.Run("tcp check widens int ports and serializes", func(t *testing.T) {
		d := loadBalancerHealthCheckDict(hcloud.LoadBalancerServiceHealthCheck{
			Protocol: hcloud.LoadBalancerServiceProtocolTCP,
			Port:     80,
			Interval: 15 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		})
		// Port and Retries are hcloud `int` — must be widened to int64 to serialize.
		assert.Equal(t, int64(80), d["port"])
		assert.Equal(t, int64(3), d["retries"])
		assert.Equal(t, float64(15), d["interval"])
		requireDictSerializes(t, d)
	})

	t.Run("http check with nested dict serializes", func(t *testing.T) {
		d := loadBalancerHealthCheckDict(hcloud.LoadBalancerServiceHealthCheck{
			Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
			Port:     443,
			Interval: 15 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
			HTTP: &hcloud.LoadBalancerServiceHealthCheckHTTP{
				Domain:      "example.com",
				Path:        "/health",
				Response:    "OK",
				StatusCodes: []string{"200", "201"},
				TLS:         true,
			},
		})
		requireDictSerializes(t, d)
	})
}

func TestLoadBalancerHealthStatusDicts(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		a := loadBalancerHealthStatusDicts(nil)
		assert.Empty(t, a)
		requireDictArraySerializes(t, a)
	})

	t.Run("widens int listenPort and serializes", func(t *testing.T) {
		a := loadBalancerHealthStatusDicts([]hcloud.LoadBalancerTargetHealthStatus{
			{ListenPort: 443, Status: hcloud.LoadBalancerTargetHealthStatusStatusHealthy},
			{ListenPort: 80, Status: hcloud.LoadBalancerTargetHealthStatusStatusUnhealthy},
		})
		require.Len(t, a, 2)
		first := a[0].(map[string]any)
		// ListenPort is hcloud `int` — must be widened to int64 to serialize.
		assert.Equal(t, int64(443), first["listenPort"])
		assert.Equal(t, "healthy", first["status"])
		requireDictArraySerializes(t, a)
	})
}
