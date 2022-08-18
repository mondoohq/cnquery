package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_DNS(t *testing.T) {
	res := x.TestQuery(t, "dns(\"mondoo.com\").mx")
	assert.NotEmpty(t, res)
}
