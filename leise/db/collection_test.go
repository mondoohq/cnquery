package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollection(t *testing.T) {
	cur := &Collection{Labels: []string{"hello", "world"}}
	assert.True(t, cur.HasLabel("world"))

	cur.RemoveLabel("world")
	assert.Equal(t, []string{"hello"}, cur.Labels)
	assert.False(t, cur.HasLabel("world"))
}
