// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mqlx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeScalars(t *testing.T) {
	var s string
	require.NoError(t, decode("hi", &s))
	assert.Equal(t, "hi", s)

	var b bool
	require.NoError(t, decode(true, &b))
	assert.True(t, b)

	var i int
	require.NoError(t, decode(int64(42), &i))
	assert.Equal(t, 42, i)

	var u uint16
	require.NoError(t, decode(int64(65535), &u))
	assert.Equal(t, uint16(65535), u)

	var f float64
	require.NoError(t, decode(int64(2), &f))
	assert.Equal(t, 2.0, f)

	// int64 fidelity: values beyond 2^53 must not lose precision.
	var big int64
	require.NoError(t, decode(int64(1<<62+1), &big))
	assert.Equal(t, int64(1<<62+1), big)
}

func TestDecodeOverflow(t *testing.T) {
	var i8 int8
	require.ErrorContains(t, decode(int64(300), &i8), "overflows")

	var u uint8
	require.ErrorContains(t, decode(int64(-1), &u), "overflows")

	var i int
	require.ErrorContains(t, decode(2.5, &i), "does not fit")

	// A float beyond int64's range must error, not produce an
	// implementation-defined int64(v) conversion.
	var big int64
	require.ErrorContains(t, decode(1e19, &big), "does not fit")
}

func TestDecodeTime(t *testing.T) {
	now := time.Now()

	var tt time.Time
	require.NoError(t, decode(&now, &tt))
	assert.True(t, now.Equal(tt))

	var pt *time.Time
	require.NoError(t, decode(&now, &pt))
	require.NotNil(t, pt)
	assert.True(t, now.Equal(*pt))
}

func TestDecodeStruct(t *testing.T) {
	src := map[string]any{
		"name":       "web-1",
		"instanceId": "i-123",
		"running":    true,
		"cpuCount":   int64(4),
		"ignored":    "nope",
	}

	type instance struct {
		ID       string `mql:"instanceId"`
		Name     string `json:"name"`
		Running  bool   // exact field-name match is case-insensitive
		CPUCount int64  `mql:"cpuCount"`
		Skipped  string `mql:"-"`
	}

	var out instance
	require.NoError(t, decode(src, &out))
	assert.Equal(t, instance{
		ID:       "i-123",
		Name:     "web-1",
		Running:  true,
		CPUCount: 4,
	}, out)
}

func TestDecodeNested(t *testing.T) {
	src := []any{
		map[string]any{
			"name": "a",
			"tags": map[string]any{"env": "prod"},
			"ips":  []any{"10.0.0.1", "10.0.0.2"},
		},
		map[string]any{
			"name": "b",
			"tags": map[string]any{},
			"ips":  []any{},
		},
	}

	type host struct {
		Name string            `mql:"name"`
		Tags map[string]string `mql:"tags"`
		IPs  []string          `mql:"ips"`
	}

	var out []host
	require.NoError(t, decode(src, &out))
	require.Len(t, out, 2)
	assert.Equal(t, "prod", out[0].Tags["env"])
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, out[0].IPs)
}

func TestDecodeEmbedded(t *testing.T) {
	type Base struct {
		ID   string `mql:"id"`
		Name string `mql:"name"`
	}
	// value embed, pointer embed, and a field that shadows an embedded one.
	type Meta struct {
		Region string `mql:"region"`
	}
	type Instance struct {
		Base
		*Meta
		Name string            `mql:"name"` // shadows Base.Name (shallower wins)
		Tags map[string]string `mql:"tags"`
	}

	src := map[string]any{
		"id":     "i-123",
		"name":   "web-1",
		"region": "us-east-1",
		"tags":   map[string]any{"env": "prod"},
	}

	var out Instance
	require.NoError(t, decode(src, &out))
	assert.Equal(t, "i-123", out.ID)         // promoted from Base
	assert.Equal(t, "web-1", out.Name)       // outer field wins over Base.Name
	assert.Equal(t, "", out.Base.Name)       // embedded one left untouched
	require.NotNil(t, out.Meta)              // nil *Meta allocated on demand
	assert.Equal(t, "us-east-1", out.Region) // promoted from *Meta
	assert.Equal(t, "prod", out.Tags["env"])
}

func TestDecodeIntoAny(t *testing.T) {
	var out any
	require.NoError(t, decode(map[string]any{"a": int64(1)}, &out))
	assert.Equal(t, map[string]any{"a": int64(1)}, out)
}

func TestDecodeNil(t *testing.T) {
	s := "unchanged"
	require.NoError(t, decode(nil, &s))
	assert.Equal(t, "unchanged", s)
}

func TestDecodeTargetErrors(t *testing.T) {
	require.ErrorContains(t, decode("x", nil), "non-nil pointer")

	var s string
	require.ErrorContains(t, decode("x", s), "non-nil pointer")

	var i int
	require.ErrorContains(t, decode("hello", &i), "cannot decode")
}
