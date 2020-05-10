package parsers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIni(t *testing.T) {
	tests := []struct {
		title   string
		content string
		res     map[string]map[string]string
	}{
		{
			"simple assignment",
			"key = value",
			map[string]map[string]string{
				"": {
					"key": "value",
				},
			},
		},
		{
			"no assignment",
			"key and value",
			map[string]map[string]string{
				"": {
					"key and value": "",
				},
			},
		},
		{
			"newline comment",
			"key\n# comment\n  # more comment\n\t# and one more\nvalue",
			map[string]map[string]string{
				"": {
					"key":   "",
					"value": "",
				},
			},
		},
		{
			"groups",
			"key\n[some group]\nkey2=value",
			map[string]map[string]string{
				"": {
					"key": "",
				},
				"some group": {
					"key2": "value",
				},
			},
		},
	}

	for i := range tests {
		cur := tests[i]
		t.Run(cur.title, func(t *testing.T) {
			res := ParseIni(cur.content)
			assert.Equal(t, cur.res, res.Fields)
		})
	}
}
