package lumi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArg2Map(t *testing.T) {
	args := []interface{}{"project", "mondoo", "zone", "us-central1-a"}

	argsmap, err := args2map(args)
	assert.Nil(t, err, "should be able to convert args to map")

	assert.Equal(t, "mondoo", (*argsmap)["project"], "extracted project arg")
	assert.Equal(t, "us-central1-a", (*argsmap)["zone"], "extracted zone arg")
}
