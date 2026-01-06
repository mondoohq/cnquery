// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntersection(t *testing.T) {
	t.Run("strings", func(t *testing.T) {
		compare := func(x, y string) bool {
			return x == y
		}
		a := []string{"a", "b", "c"}
		b := []string{"b", "c", "d"}
		c := []string{"e", "f"}
		d := []string{}
		inter := Intersection(a, b, compare)
		assert.Equal(t, []string{"b", "c"}, inter)
		inter = Intersection(a, c, compare)
		assert.Equal(t, []string{}, inter)
		inter = Intersection(a, d, compare)
		assert.Equal(t, []string{}, inter)
	})

	t.Run("ints", func(t *testing.T) {
		compare := func(x, y int) bool {
			return x == y
		}
		a := []int{1, 2, 3, 4}
		b := []int{3, 4, 5, 6}
		c := []int{7, 8}
		d := []int{}
		inter := Intersection(a, b, compare)
		assert.Equal(t, []int{3, 4}, inter)
		inter = Intersection(a, c, compare)
		assert.Equal(t, []int{}, inter)
		inter = Intersection(a, d, compare)
		assert.Equal(t, []int{}, inter)
	})

	t.Run("structs", func(t *testing.T) {
		type person struct {
			name string
			age  int
		}
		compare := func(x, y person) bool {
			return x.name == y.name && x.age == y.age
		}
		a := []person{
			{name: "Alice", age: 30},
			{name: "Bob", age: 25},
			{name: "Charlie", age: 35},
		}
		b := []person{
			{name: "Bob", age: 25},
			{name: "David", age: 40},
		}
		c := []person{
			{name: "Eve", age: 28},
		}
		d := []person{}
		inter := Intersection(a, b, compare)
		assert.Equal(t, []person{{name: "Bob", age: 25}}, inter)
		inter = Intersection(a, c, compare)
		assert.Equal(t, []person{}, inter)
		inter = Intersection(a, d, compare)
		assert.Equal(t, []person{}, inter)
	})
}
