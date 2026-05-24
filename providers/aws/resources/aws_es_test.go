// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkStrings(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		size int
		want [][]string
	}{
		{
			name: "nil input",
			in:   nil,
			size: 5,
			want: nil,
		},
		{
			name: "empty input",
			in:   []string{},
			size: 5,
			want: nil,
		},
		{
			name: "single element below batch",
			in:   []string{"a"},
			size: 5,
			want: [][]string{{"a"}},
		},
		{
			name: "exact batch boundary",
			in:   []string{"a", "b", "c", "d", "e"},
			size: 5,
			want: [][]string{{"a", "b", "c", "d", "e"}},
		},
		{
			name: "one over boundary produces partial second chunk",
			in:   []string{"a", "b", "c", "d", "e", "f"},
			size: 5,
			want: [][]string{{"a", "b", "c", "d", "e"}, {"f"}},
		},
		{
			name: "two full batches",
			in:   []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			size: 5,
			want: [][]string{
				{"a", "b", "c", "d", "e"},
				{"f", "g", "h", "i", "j"},
			},
		},
		{
			name: "uneven trailing chunk",
			in:   []string{"a", "b", "c", "d", "e", "f", "g"},
			size: 3,
			want: [][]string{
				{"a", "b", "c"},
				{"d", "e", "f"},
				{"g"},
			},
		},
		{
			name: "size of one",
			in:   []string{"a", "b", "c"},
			size: 1,
			want: [][]string{{"a"}, {"b"}, {"c"}},
		},
		{
			name: "size larger than input",
			in:   []string{"a", "b"},
			size: 100,
			want: [][]string{{"a", "b"}},
		},
		{
			name: "non-positive size returns whole slice",
			in:   []string{"a", "b", "c"},
			size: 0,
			want: [][]string{{"a", "b", "c"}},
		},
		{
			name: "negative size returns whole slice",
			in:   []string{"a", "b", "c"},
			size: -1,
			want: [][]string{{"a", "b", "c"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, chunkStrings(tc.in, tc.size))
		})
	}
}

// TestChunkStringsCoversInput sanity-checks that for the AWS ES batch size,
// every input element appears exactly once across chunks, in order.
func TestChunkStringsCoversInput(t *testing.T) {
	for n := 0; n <= 23; n++ {
		in := make([]string, n)
		for i := range in {
			in[i] = string(rune('a' + i))
		}
		chunks := chunkStrings(in, esDescribeBatchSize)

		flat := []string{}
		for _, c := range chunks {
			assert.LessOrEqual(t, len(c), esDescribeBatchSize, "chunk exceeded batch size for n=%d", n)
			flat = append(flat, c...)
		}
		assert.Equal(t, in, flat, "round-trip failed for n=%d", n)
	}
}
