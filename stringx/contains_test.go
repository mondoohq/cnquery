package stringx_test

import (
	"go.mondoo.io/mondoo/stringx"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	assert.True(t, stringx.Contains([]string{"ab", "aa"}, "ab"))
	assert.False(t, stringx.Contains([]string{"ab", "aa"}, "a"))
	assert.False(t, stringx.Contains([]string{"ab", "aa"}, "bs"))
	assert.True(t, stringx.Contains([]string{"hello", "world"}, "world"))
	assert.True(t, stringx.Contains([]string{"hello", "world"}, "hello"))
	assert.False(t, stringx.Contains([]string{"hello", "world"}, "john"))
}

func TestRemoveEmpty(t *testing.T) {
	assert.Equal(t, []string{"aa"}, stringx.RemoveEmpty([]string{"", "aa"}))
	assert.Equal(t, []string{"aa"}, stringx.RemoveEmpty([]string{"aa", ""}))
	assert.Equal(t, []string{"aa", "ab"}, stringx.RemoveEmpty([]string{"", "aa", "", "ab", ""}))
}
