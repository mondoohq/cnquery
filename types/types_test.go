// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package types

import (
	"testing"

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

func TestType_EmptyTypeAccessorsDoNotPanic(t *testing.T) {
	empty := Type("")
	bareArray := ArrayLike

	assert.NotPanics(t, func() {
		assert.Equal(t, Unset, empty.Underlying())
	})
	assert.NotPanics(t, func() {
		assert.Equal(t, NoType, empty.Child())
	})
	assert.NotPanics(t, func() {
		assert.False(t, empty.IsArray())
	})
	assert.NotPanics(t, func() {
		assert.False(t, empty.IsFunction())
	})

	// A bare ArrayLike (byteArray with no element type) reports an empty
	// child type rather than panicking when sliced.
	assert.NotPanics(t, func() {
		assert.Equal(t, NoType, bareArray.Child())
	})
}
