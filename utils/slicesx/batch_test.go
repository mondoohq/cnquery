// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatch(t *testing.T) {
	slice := []string{}
	for i := 0; i < 100; i++ {
		slice = append(slice, fmt.Sprintf("item-%d", i))
	}

	batches := Batch(slice, 10)
	assert.Len(t, batches, 10)

	flattenedBatches := []string{}
	for _, batch := range batches {
		flattenedBatches = append(flattenedBatches, batch...)
	}
	assert.Equal(t, slice, flattenedBatches)
}

func TestBatch_Uneven(t *testing.T) {
	slice := []string{}
	for i := 0; i < 101; i++ {
		slice = append(slice, fmt.Sprintf("item-%d", i))
	}

	batches := Batch(slice, 10)
	assert.Len(t, batches, 11)

	flattenedBatches := []string{}
	for _, batch := range batches {
		flattenedBatches = append(flattenedBatches, batch...)
	}
	assert.Equal(t, slice, flattenedBatches)
}
