package parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIni(t *testing.T) {
	tests := []struct {
		title   string
		content string
		res     map[string]interface{}
	}{
		{
			"simple assignment",
			"key = value",
			map[string]interface{}{
				"": map[string]interface{}{
					"key": "value",
				},
			},
		},
		{
			"no assignment",
			"key and value",
			map[string]interface{}{
				"": map[string]interface{}{
					"key and value": "",
				},
			},
		},
		{
			"newline comment",
			"key\n# comment\n  # more comment\n\t# and one more\nvalue",
			map[string]interface{}{
				"": map[string]interface{}{
					"key":   "",
					"value": "",
				},
			},
		},
		{
			"groups",
			"key\n[some group]\nkey2=value",
			map[string]interface{}{
				"": map[string]interface{}{
					"key": "",
				},
				"some group": map[string]interface{}{
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
		res     map[string]interface{}
	}{
		{
			"simple assignment",
			"key value",
			map[string]interface{}{
				"": map[string]interface{}{
					"key": "value",
				},
			},
		},
		{
			"no assignment",
			"keykey",
			map[string]interface{}{
				"": map[string]interface{}{
					"keykey": "",
				},
			},
		},
		{
			"newline comment",
			"key\n# comment\n  # more comment\n\t# and one more\nvalue",
			map[string]interface{}{
				"": map[string]interface{}{
					"key":   "",
					"value": "",
				},
			},
		},
		{
			"groups",
			"key\n[some group]\nkey2 value",
			map[string]interface{}{
				"": map[string]interface{}{
					"key": "",
				},
				"some group": map[string]interface{}{
					"key2": "value",
				},
			},
		},
		{
			"tabs",
			"key\tvalue",
			map[string]interface{}{
				"": map[string]interface{}{
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
