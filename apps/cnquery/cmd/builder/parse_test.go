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
