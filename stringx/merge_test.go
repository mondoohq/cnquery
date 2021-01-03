package stringx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerge(t *testing.T) {

	test1 := "abc def\nhfr tre"
	test2 := "123 456\n789 123"

	actual := MergeSideBySide(test1, test2)

	expected := "abc def123 456\nhfr tre789 123\n"
	assert.Equal(t, actual, expected)
}
