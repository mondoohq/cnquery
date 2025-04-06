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

func TestParseXML(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "parse.xml(content: '<root />').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{}},
		},
		{
			Code:        "parse.xml(content: '<root>\n\t\t\n</root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{}},
		},
		{
			Code:        "parse.xml(content: '<root>\n\tworld\n</root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": "world"},
		},
		{
			Code:        "parse.xml(content: '<root>\n\tworld\n\twide\n</root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": "world\n\twide"},
		},
		{
			Code:        "parse.xml(content: '<root><box /></root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{"box": map[string]any{}}},
		},
		{
			Code:        "parse.xml(content: '<root><box>world</box></root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{"box": "world"}},
		},
		{
			Code:        "parse.xml(content: '<root><box>hello</box><box>world</box></root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{"box": []any{
				"hello",
				"world",
			}}},
		},
		{
			Code:        "parse.xml(content: '<root><box><hello a=\"1\"/></box><box><world b=\"2\">1<c>3</c>4</world></box><box>🌎</box></root>').params",
			ResultIndex: 0,
			Expectation: map[string]any{"root": map[string]any{"box": []any{
				map[string]any{"hello": map[string]any{"@a": "1"}},
				map[string]any{"world": map[string]any{
					"@b":     "2",
					"c":      "3",
					"__text": "1\n4",
				}},
				"🌎",
			}}},
		},
	})
}
