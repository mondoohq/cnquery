// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	list := []struct {
		T             Type
		ExpectedLabel string
	}{
		{T: Unset, ExpectedLabel: "unset"},
		{T: Any, ExpectedLabel: "any"},
		{T: Nil, ExpectedLabel: "null"},
		{T: Ref, ExpectedLabel: "ref"},
		{T: Bool, ExpectedLabel: "bool"},
		{T: Int, ExpectedLabel: "int"},
		{T: Float, ExpectedLabel: "float"},
		{T: String, ExpectedLabel: "string"},
		{T: Regex, ExpectedLabel: "regex"},
		{T: Time, ExpectedLabel: "time"},
		{T: Dict, ExpectedLabel: "dict"},
		{T: Score, ExpectedLabel: "score"},
		{T: Block, ExpectedLabel: "block"},
		{T: Empty, ExpectedLabel: "empty"},
		{T: Version, ExpectedLabel: "version"},
		{T: IP, ExpectedLabel: "ip"},
		{T: Array(String), ExpectedLabel: "[]string"},
		{T: Map(String, String), ExpectedLabel: "map[string]string"},
		{T: Resource("mockresource"), ExpectedLabel: "mockresource"},
		{T: Function('f', []Type{String, Int}), ExpectedLabel: "func()"},
	}

	for i := range list {
		test := list[i]

		// test for human friendly name
		assert.Equal(t, test.ExpectedLabel, test.T.Label())
	}
}

func TestEqual_NilOperands(t *testing.T) {
	// A null array element surfaces as a nil interface value. The Equal
	// comparators must treat nil safely (nil == nil, nil \!= value) instead
	// of panicking on the type assertion, which would crash the whole scan.
	cases := []struct {
		typ   Type
		value any
	}{
		{Bool, true},
		{Int, int64(1)},
		{Float, 1.5},
		{String, "x"},
		{Regex, "x"},
		{Score, int32(1)},
		{Time, &time.Time{}},
	}
	for _, c := range cases {
		eq, ok := Equal[c.typ]
		assert.True(t, ok, c.typ.Label())
		assert.False(t, eq(c.value, nil), "%s: value == nil", c.typ.Label())
		assert.False(t, eq(nil, c.value), "%s: nil == value", c.typ.Label())
		assert.True(t, eq(nil, nil), "%s: nil == nil", c.typ.Label())
	}
}
