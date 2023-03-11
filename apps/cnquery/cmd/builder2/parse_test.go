package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetParser(t *testing.T) {
	tests := []struct {
		value  string
		target target
	}{
		{
			value:  "user@host:10",
			target: target{Hostname: "host", Username: "user", Path: "", Port: 10},
		},
		{
			value:  "ssh://user@host:10",
			target: target{Hostname: "host", Username: "user", Path: "", Port: 10},
		},
		{
			value:  "user@host",
			target: target{Hostname: "host", Username: "user", Path: "", Port: 0},
		},
		{
			value:  "user@admin@host",
			target: target{Hostname: "host", Username: "user@admin", Path: "", Port: 0},
		},
	}

	for i := range tests {
		res, err := parseTarget(tests[i].value)
		require.NoError(t, err)
		assert.Equal(t, tests[i].target, res, tests[i].value)
	}
}

func TestParseIPv6(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "//192.168.123.22",
			expected: "//192.168.123.22",
		},
		{
			input:    "ssh://user@192.168.123.22:22",
			expected: "ssh://user@192.168.123.22:22",
		},
		{
			input:    "//user@192.168.123.22:22",
			expected: "//user@192.168.123.22:22",
		},
		{
			input:    "ssh://user@[fe80::a3c4:928f:d918:22]:22",
			expected: "ssh://user@[fe80::a3c4:928f:d918:22]:22",
		},
		{
			input:    "//user@[fe80::a3c4:928f:d918:22]:22",
			expected: "//user@[fe80::a3c4:928f:d918:22]:22",
		},
		{
			input:    "//user@fe80::a3c4:d918:22",
			expected: "//user@[fe80::a3c4:d918:22]",
		},
		{
			input:    "//user@fe80::a3c4:d918:22:22", // no abiliy to understand user might have meant port 22!
			expected: "//user@[fe80::a3c4:d918:22:22]",
		},
		{
			input:    "//fe80::a3c4:d918:22",
			expected: "//[fe80::a3c4:d918:22]",
		},
		{
			input:    "//[fe80::a3c4:d918:22]:22",
			expected: "//[fe80::a3c4:d918:22]:22",
		},
	}

	for _, test := range tests {
		res := addIPv6Brackets(test.input)
		assert.Equal(t, test.expected, res, "unexpected parsing result during ipv6 handling")
	}
}
