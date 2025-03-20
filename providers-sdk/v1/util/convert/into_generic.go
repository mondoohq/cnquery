// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

// Func defines a function type that converts an element of type T to V.
type Func[T any, V any] func(T) V

// Into converts a slice of type `T` to a slice of type `V` by applying the
// convert function `Func`.
//
// Example usage:
//
//  1. Convert a slice of integers to a slice of strings:
//
//     ints := []int{1, 2, 3}
//     strs := Into(ints, func(i int) string {
//     return fmt.Sprintf("%d", i)
//     })
//
//     // Output: []string{"1", "2", "3"}
//
//  2. Convert a slice of structs into a slice of strings:
//
//     type Person struct {
//     Name string
//     Age  int
//     }
//
//     people := []Person{
//     {Name: "Alice", Age: 30},
//     {Name: "Bob", Age: 25},
//     }
//
//     names := Into(people, func(p Person) string {
//     return p.Name
//     })
//
//     // Output: []string{"Alice", "Bob"}
func Into[T any, V any](sliceT []T, convertFn Func[T, V]) []V {
	sliceV := make([]V, len(sliceT))
	for i, item := range sliceT {
		sliceV[i] = convertFn(item)
	}
	return sliceV
}
