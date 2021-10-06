package stringx_test

import (
	"testing"

	"go.mondoo.io/mondoo/stringx"

	"github.com/stretchr/testify/assert"
)

func TestDedupStringArray(t *testing.T) {
	arr := []string{"a", "a", "b", "b", "c"}
	assert.ElementsMatch(t, []string{"a", "b", "c"}, stringx.DedupStringArray(arr))
}
