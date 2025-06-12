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

func TestParseYamlParams(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
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
		{
			Code:        `parse.yaml(content: "").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{},
		},
		{
			Code:        `parse.yaml(content: "---\nname: single-doc\nversion: 1.2").params`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{
				"name":    "single-doc",
				"version": float64(1.2),
			},
		},
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

func TestParseYamlDocuments(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        `parse.yaml(content: "").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{},
		},
		{
			Code:        `parse.yaml(content: "simple: test").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{
					"simple": "test",
				},
			},
		},
		{
			Code:        `parse.yaml(content: "---\nname: single-doc\nversion: 1.2").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{
					"name":    "single-doc",
					"version": float64(1.2),
				},
			},
		},
		{
			Code:        `parse.yaml(content: "name: trailing-doc\nversion: 1.2\n---").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{
					"name":    "trailing-doc",
					"version": float64(1.2),
				},
			},
		},
		{
			Code:        `parse.yaml(content: "---\nname: wrapped-doc\nversion: 1.2\n---").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{
					"name":    "wrapped-doc",
					"version": float64(1.2),
				},
			},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{"name": "doc1"},
				map[string]interface{}{"name": "doc2"},
			},
		},
		{
			Code:        `parse.yaml(content: "---\nname: doc1\n---\nname: doc2").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{"name": "doc1"},
				map[string]interface{}{"name": "doc2"},
			},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2\n---").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{"name": "doc1"},
				map[string]interface{}{"name": "doc2"},
			},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2\n---\nname: doc3").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{"name": "doc1"},
				map[string]interface{}{"name": "doc2"},
				map[string]interface{}{"name": "doc3"},
			},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").documents[0]`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{"name": "doc1"},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2").documents[1]`,
			ResultIndex: 0,
			Expectation: map[string]interface{}{"name": "doc2"},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\n\n---\nname: doc2").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{"name": "doc1"},
				map[string]interface{}{"name": "doc2"},
			},
		},
		{
			Code:        `parse.yaml(content: "apiVersion: v1\nkind: Service\n---\napiVersion: apps/v1\nkind: Deployment").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{
				map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
				},
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
				},
			},
		},
		{
			Code:        `parse.yaml(content: "---").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{},
		},
		{
			Code:        `parse.yaml(content: "---\n---").documents`,
			ResultIndex: 0,
			Expectation: []interface{}{},
		},
		{
			Code:        `parse.yaml(content: "name: doc1\n---\nname: doc2\n---\nname: doc3").documents.length`,
			ResultIndex: 0,
			Expectation: int64(3),
		},
	})
}
