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

func TestParseYaml(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		// Basic single document tests
		{
			Code:        `parse.yaml(content: "simple: test").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"simple": "test",
			},
		},
		{
			Code:        `parse.yaml(content: "number: 42").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"number": float64(42),
			},
		},
		{
			Code:        `parse.yaml(content: "enabled: true").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"enabled": true,
			},
		},
		{
			Code:        `parse.yaml(content: "parent:\n  child: value").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"parent": map[string]interface{}{
					"child": "value",
				},
			},
		},

		// Empty content
		{
			Code:        `parse.yaml(content: "").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{},
		},

		// Single document with leading --- (common pattern)
		{
			Code:        `parse.yaml(content: "---\nname: single-doc\nversion: 1.2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"name":    "single-doc",
				"version": float64(1.2),
			},
		},

		// Single document with trailing ---
		{
			Code:        `parse.yaml(content: "name: trailing-doc\nversion: 1.2\n---").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"name":    "trailing-doc",
				"version": float64(1.2),
			},
		},

		// Single document with both leading and trailing ---
		{
			Code:        `parse.yaml(content: "---\nname: wrapped-doc\nversion: 1.2\n---").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"name":    "wrapped-doc",
				"version": float64(1.2),
			},
		},

		// True multi-document YAML
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{"name": "doc1"},
				"1": map[string]interface{}{"name": "doc2"},
			},
		},

		// Multi-document with leading ---
		{
			Code:        `parse.yaml(content: "---\nname: doc1\n---\nname: doc2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{"name": "doc1"},
				"1": map[string]interface{}{"name": "doc2"},
			},
		},

		// Multi-document with trailing ---
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2\n---").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{"name": "doc1"},
				"1": map[string]interface{}{"name": "doc2"},
			},
		},

		// Three documents
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2\n---\nname: doc3").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{"name": "doc1"},
				"1": map[string]interface{}{"name": "doc2"},
				"2": map[string]interface{}{"name": "doc3"},
			},
		},

		// Access specific document from multi-document
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").params["0"]`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{"name": "doc1"},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").params["1"]`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{"name": "doc2"},
		},

		// Multi-document with empty documents (should be skipped)
		{
			Code:        `parse.yaml(content: "name: doc1\n---\n\n---\nname: doc2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{"name": "doc1"},
				"1": map[string]interface{}{"name": "doc2"},
			},
		},

		// Complex nested structures
		{
			Code:        `parse.yaml(content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\ndata:\n  key1: value1\n  key2: value2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "test",
				},
				"data": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},

		// Arrays in YAML
		{
			Code:        `parse.yaml(content: "items:\n  - name: item1\n  - name: item2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "item1"},
					map[string]interface{}{"name": "item2"},
				},
			},
		},

		// Multi-document with different structures
		{
			Code:        `parse.yaml(content: "apiVersion: v1\nkind: Service\n---\napiVersion: apps/v1\nkind: Deployment").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"0": map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
				},
				"1": map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
				},
			},
		},

		// Edge case: just separators
		{
			Code:        `parse.yaml(content: "---").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{},
		},

		// Edge case: multiple separators only
		{
			Code:        `parse.yaml(content: "---\n---").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{},
		},

		// Kubernetes-style manifest
		{
			Code:        `parse.yaml(content: "---\napiVersion: v1\nkind: Pod\nmetadata:\n  name: test-pod\nspec:\n  containers:\n  - name: test\n    image: nginx").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "test-pod",
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "test",
							"image": "nginx",
						},
					},
				},
			},
		},
	})
}
