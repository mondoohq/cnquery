package checksums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksums(t *testing.T) {
	c := New
	c = c.Add("hello")
	assert.Equal(t, "C72qgEbYMKQ=", c.String())
	c = c.Add("world")
	assert.Equal(t, "gVVKkl4x2RA=", c.String())
}
