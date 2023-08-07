package stringx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/utils/stringx"
)

func TestMerge(t *testing.T) {
	test1 := "abc def\nhfr tre"
	test2 := "123 456\n789 123"

	actual := stringx.MergeSideBySide(test1, test2)

	expected := "abc def123 456\nhfr tre789 123\n"
	assert.Equal(t, actual, expected)
}
