// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIni(t *testing.T) {
	tests := []struct {
		title   string
		content string
		res     map[string]any
	}{
		{
			"simple assignment",
			"key = value",
			map[string]any{
				"": map[string]any{
					"key": "value",
				},
			},
		},
		{
			"no assignment",
			"key and value",
			map[string]any{
				"": map[string]any{
					"key and value": "",
				},
			},
		},
		{
			"newline comment",
			"key\n# comment\n  # more comment\n\t# and one more\nvalue",
			map[string]any{
				"": map[string]any{
					"key":   "",
					"value": "",
				},
			},
		},
		{
			"groups",
			"key\n[some group]\nkey2=value",
			map[string]any{
				"": map[string]any{
					"key": "",
				},
				"some group": map[string]any{
					"key2": "value",
				},
			},
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			res := ParseIni(cur.content, "=")
			assert.Equal(t, cur.res, res.Fields)
		})
	}
}

func TestIni_SpaceDelim(t *testing.T) {
	tests := []struct {
		title   string
		content string
		res     map[string]any
	}{
		{
			"simple assignment",
			"key value",
			map[string]any{
				"": map[string]any{
					"key": "value",
				},
			},
		},
		{
			"no assignment",
			"keykey",
			map[string]any{
				"": map[string]any{
					"keykey": "",
				},
			},
		},
		{
			"newline comment",
			"key\n# comment\n  # more comment\n\t# and one more\nvalue",
			map[string]any{
				"": map[string]any{
					"key":   "",
					"value": "",
				},
			},
		},
		{
			"groups",
			"key\n[some group]\nkey2 value",
			map[string]any{
				"": map[string]any{
					"key": "",
				},
				"some group": map[string]any{
					"key2": "value",
				},
			},
		},
		{
			"tabs",
			"key\tvalue",
			map[string]any{
				"": map[string]any{
					"key": "value",
				},
			},
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			res := ParseIni(cur.content, " ")
			assert.Equal(t, cur.res, res.Fields)
		})
	}
}

func TestJournalD(t *testing.T) {
	data := `
[Journal]
Storage=auto
Compress=yes
#Seal=yes
`
	res := ParseIni(data, "=")
	assert.Equal(t, map[string]any{
		"Journal": map[string]any{
			"Storage":  "auto",
			"Compress": "yes",
		},
	}, res.Fields)
}
