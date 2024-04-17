// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestParsePlist(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.plist('/dummy.plist').params['allowdownloadsignedenabled']",
			ResultIndex: 0,
			// validates that the output is not uint64
			Expectation: float64(1),
		},
	})
}

func TestParseJson(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.json(content: '{\"a\": 1}').params",
			ResultIndex: 0,
			Expectation: map[string]interface{}{"a": float64(1)},
		},
		{
			Code:        "parse.json(content: '[{\"a\": 1}]').params[0]",
			ResultIndex: 0,
			Expectation: map[string]interface{}{"a": float64(1)},
		},
	})
}
