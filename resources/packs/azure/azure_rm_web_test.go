package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	assert.False(t, isPlatformEol("node", "12-lts"))
	assert.False(t, isPlatformEol("node", "10-lts"))
	assert.True(t, isPlatformEol("node", "11.1"))
	assert.True(t, isPlatformEol("node", "6.1"))
}
