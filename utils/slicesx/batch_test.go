package slicesx

import (
	"fmt"
	"testing"

	"github.com/tj/assert"
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
