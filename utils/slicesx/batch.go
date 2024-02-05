// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

func Batch[T any](list []T, batchSize int) [][]T {
	var res [][]T
	for i := 0; i < len(list); i += batchSize {
		end := i + batchSize
		if end > len(list) {
			end = len(list)
		}
		res = append(res, list[i:end])
	}
	return res
}
