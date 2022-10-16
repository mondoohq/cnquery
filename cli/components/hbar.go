package components

import (
	"math"
	"strings"
)

// Hbar creates a horizontal bar up to len characters with a
// length proportional to percent, in range of [0, 100]
func Hbar(len uint32, percent float32) string {
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}
	segments := math.Round(float64(len) * (float64(percent) / 100))

	if segments == 0 {
		return ""
	}
	return strings.Repeat("█", int(segments))
}
