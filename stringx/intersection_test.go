package stringx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntersection(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"b", "c", "d", "f"}

	actual := Intersection(a, b)
	expected := []string{"b", "c"}
	assert.ElementsMatch(t, actual, expected)
}

func TestIntersectionNoOverlap(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"d", "f"}

	actual := Intersection(a, b)
	assert.Empty(t, actual)
}
