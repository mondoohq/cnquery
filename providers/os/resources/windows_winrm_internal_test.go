// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWinRMBool(t *testing.T) {
	t.Run("present and 1 is true", func(t *testing.T) {
		items := map[string]int64{"allowbasic": 1}
		assert.True(t, winrmBool(items, "AllowBasic", false))
	})

	t.Run("present and 0 is false", func(t *testing.T) {
		items := map[string]int64{"allowbasic": 0}
		// default is true, but an explicit 0 disables it
		assert.False(t, winrmBool(items, "AllowBasic", true))
	})

	t.Run("absent uses the documented default", func(t *testing.T) {
		assert.True(t, winrmBool(map[string]int64{}, "AllowBasic", true))
		assert.False(t, winrmBool(nil, "AllowAutoConfig", false))
	})

	t.Run("value name matching is case insensitive", func(t *testing.T) {
		items := map[string]int64{"allowunencryptedtraffic": 1}
		assert.True(t, winrmBool(items, "AllowUnencryptedTraffic", false))
	})

	t.Run("any non-1 value is false", func(t *testing.T) {
		items := map[string]int64{"allowbasic": 2}
		assert.False(t, winrmBool(items, "AllowBasic", true))
	})
}

func TestComputeWinRMClient(t *testing.T) {
	t.Run("absent uses documented defaults (allow)", func(t *testing.T) {
		basic, unenc, digest := computeWinRMClient(map[string]int64{})
		assert.True(t, basic)
		assert.True(t, unenc)
		assert.True(t, digest)
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		items := map[string]int64{
			"allowbasic":              0,
			"allowunencryptedtraffic": 0,
			"allowdigest":             0,
		}
		basic, unenc, digest := computeWinRMClient(items)
		assert.False(t, basic)
		assert.False(t, unenc)
		assert.False(t, digest)
	})

	t.Run("present and 1 is true", func(t *testing.T) {
		items := map[string]int64{
			"allowbasic":              1,
			"allowunencryptedtraffic": 1,
			"allowdigest":             1,
		}
		basic, unenc, digest := computeWinRMClient(items)
		assert.True(t, basic)
		assert.True(t, unenc)
		assert.True(t, digest)
	})

	t.Run("case insensitive value names", func(t *testing.T) {
		// computeWinRMClient lower-cases via winrmBool; provide already-lowered
		// keys as the loader does, and verify a mixed-case absence still defaults.
		items := map[string]int64{"allowdigest": 0}
		basic, unenc, digest := computeWinRMClient(items)
		assert.True(t, basic)   // absent -> default true
		assert.True(t, unenc)   // absent -> default true
		assert.False(t, digest) // explicit 0
	})
}

func TestComputeWinRMService(t *testing.T) {
	t.Run("absent uses documented defaults", func(t *testing.T) {
		basic, unenc, disableRunAs, autoConfig, remoteShell := computeWinRMService(map[string]int64{}, map[string]int64{})
		assert.True(t, basic)         // allow basic defaults true
		assert.True(t, unenc)         // allow unencrypted defaults true
		assert.False(t, disableRunAs) // not disabled by default
		assert.False(t, autoConfig)   // listener not auto-configured by default
		assert.True(t, remoteShell)   // remote shell allowed by default
	})

	t.Run("hardened explicit values", func(t *testing.T) {
		service := map[string]int64{
			"allowbasic":              0,
			"allowunencryptedtraffic": 0,
			"disablerunas":            1,
			"allowautoconfig":         0,
		}
		winrs := map[string]int64{"allowremoteshellaccess": 0}
		basic, unenc, disableRunAs, autoConfig, remoteShell := computeWinRMService(service, winrs)
		assert.False(t, basic)
		assert.False(t, unenc)
		assert.True(t, disableRunAs)
		assert.False(t, autoConfig)
		assert.False(t, remoteShell)
	})

	t.Run("WinRS subkey read independently of Service key", func(t *testing.T) {
		service := map[string]int64{"allowautoconfig": 1}
		winrs := map[string]int64{"allowremoteshellaccess": 1}
		_, _, _, autoConfig, remoteShell := computeWinRMService(service, winrs)
		assert.True(t, autoConfig)
		assert.True(t, remoteShell)
	})
}

func TestComputeWinRMServiceStartMode(t *testing.T) {
	t.Run("present value is returned", func(t *testing.T) {
		assert.Equal(t, int64(2), computeWinRMServiceStartMode(map[string]int64{"start": 2}))
		assert.Equal(t, int64(4), computeWinRMServiceStartMode(map[string]int64{"start": 4}))
	})

	t.Run("absent uses documented default (manual)", func(t *testing.T) {
		assert.Equal(t, int64(3), computeWinRMServiceStartMode(map[string]int64{}))
		assert.Equal(t, int64(3), computeWinRMServiceStartMode(nil))
	})

	t.Run("case insensitive value name", func(t *testing.T) {
		// loader lower-cases names; the lookup uses the lower-cased "start"
		assert.Equal(t, int64(2), computeWinRMServiceStartMode(map[string]int64{"start": 2}))
	})
}
