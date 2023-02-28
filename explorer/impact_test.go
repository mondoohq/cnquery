package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImpact(t *testing.T) {
	impact := Impact{Value: 0}
	assert.Equal(t, "none", impact.Rating())
	impact = Impact{Value: 10}
	assert.Equal(t, "low", impact.Rating())
	impact = Impact{Value: 50}
	assert.Equal(t, "medium", impact.Rating())
	impact = Impact{Value: 80}
	assert.Equal(t, "high", impact.Rating())
	impact = Impact{Value: 100}
	assert.Equal(t, "critical", impact.Rating())
	impact = Impact{Value: -1}
	assert.Equal(t, "unknown", impact.Rating())
}
